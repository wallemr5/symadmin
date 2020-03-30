package v2

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
	helmenv "k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/release"
	hapirelease "k8s.io/helm/pkg/proto/hapi/release"
	rls "k8s.io/helm/pkg/proto/hapi/services"
	"k8s.io/klog"
)

// DefaultNamespace default namespace
const DefaultNamespace = "default"

// SystemNamespace K8s system namespace
const SystemNamespace = "kube-system"

//
const versionAll = "all"

const maxCompressedDataSize = 10485760
const maxDataSize = 10485760

// DefaultInstallOptions contains th default install options used for creating a new helm release
// nolint: gochecknoglobals
var DefaultInstallOptions = []helm.InstallOption{
	helm.InstallReuseName(true),
	helm.InstallDisableHooks(false),
	helm.InstallTimeout(300),
	helm.InstallWait(false),
	helm.InstallDryRun(false),
}

// ReleaseNotFoundError is returned when a Helm related operation is executed on
// a release (helm release) that doesn't exists
type ReleaseNotFoundError struct {
	HelmError error
}

func (e *ReleaseNotFoundError) Error() string {
	return fmt.Sprintf("release not found: %s", e.HelmError)
}

type chartDataIsTooBigError struct {
	size int64
}

func (e *chartDataIsTooBigError) Error() string {
	return "chart data is too big"
}

func (e *chartDataIsTooBigError) Context() []interface{} {
	return []interface{}{"maxAllowedSize", maxCompressedDataSize, "size", e.size}
}

// DownloadFile download file/unzip and untar and store it in memory
func DownloadFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	compressedContent := new(bytes.Buffer)

	if resp.ContentLength > maxCompressedDataSize {
		return nil, errors.WithStack(&chartDataIsTooBigError{resp.ContentLength})
	}

	_, copyErr := io.CopyN(compressedContent, resp.Body, maxCompressedDataSize)
	if copyErr != nil && copyErr != io.EOF {
		return nil, errors.Wrap(err, "failed to read from chart response")
	}

	gzf, err := gzip.NewReader(compressedContent)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open chart gzip archive")
	}
	defer gzf.Close()

	tarContent := new(bytes.Buffer)
	_, copyErr = io.CopyN(tarContent, gzf, maxDataSize)
	if copyErr != nil && copyErr != io.EOF {
		return nil, errors.Wrap(copyErr, "failed to read from chart data archive")
	}

	return tarContent.Bytes(), nil
}

// GetChartFile fetches a file from the chart.
func GetChartFile(file []byte, fileName string) (string, error) {
	tarReader := tar.NewReader(bytes.NewReader(file))

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return "", err
		}

		// We search for explicit path and the root directory is unknown.
		// Apply regexp (<anything>/filename prevent false match like /root_dir/chart/abc/README.md
		match, _ := regexp.MatchString("^([^/]*)/"+fileName+"$", header.Name)
		if match {
			fileContent, err := ioutil.ReadAll(tarReader)
			if err != nil {
				return "", err
			}

			if filepath.Ext(fileName) == ".md" {
				klog.Infof("Security transform: %s", fileName)
				klog.Infof("Origin: %s", fileContent)

				fileContent = bluemonday.UGCPolicy().SanitizeBytes(fileContent)
			}

			base64File := base64.StdEncoding.EncodeToString(fileContent)

			return base64File, nil
		}
	}

	return "", nil
}

// DeleteAllRelease deletes all Helm release
func DeleteAllRelease(hClient *Client) error {
	klog.Info("getting releases....")
	filter := ""
	releaseResp, err := ListReleases(filter, hClient)
	if err != nil {
		return errors.Wrap(err, "failed to get releases")
	}

	if releaseResp != nil {
		// the returned release items are unique by release name and status
		// e.g. release name = release1, status = PENDING_UPGRADE
		//      release name = release1, status = DEPLOYED
		//
		// we need only the release name for deleting a release
		deletedReleases := make(map[string]bool)
		for _, r := range releaseResp.Releases {
			if _, ok := deletedReleases[r.Name]; !ok {
				klog.Infof("deleting release, name:%s", r.Name)
				err := DeleteRelease(r.Name, hClient)
				if err != nil {
					return errors.Wrapf(err, "failed to delete release: %s", r.Name)
				}
				deletedReleases[r.Name] = true

				klog.Infof("release successfully deleted, name:%s", r.Name)
			}
		}
	}
	return nil
}

// ListReleases lists Helm releases
func ListReleases(filter string, hClient *Client) (*rls.ListReleasesResponse, error) {
	ops := []helm.ReleaseListOption{
		helm.ReleaseListSort(int32(rls.ListSort_LAST_RELEASED)),
		helm.ReleaseListOrder(int32(rls.ListSort_DESC)),
		helm.ReleaseListStatuses([]release.Status_Code{
			release.Status_DEPLOYED,
			release.Status_FAILED,
			release.Status_DELETING,
			release.Status_PENDING_INSTALL,
			release.Status_PENDING_UPGRADE,
			release.Status_PENDING_ROLLBACK}),
		// helm.ReleaseListLimit(limit),
		// helm.ReleaseListNamespace(""),
	}
	if filter != "" {
		klog.V(4).Infof("Apply filters: %s", filter)
		ops = append(ops, helm.ReleaseListFilter(filter))
	}

	resp, err := hClient.ListReleases(ops...)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func ListToReleasesMeta(list *rls.ListReleasesResponse) []*GetReleaseResponse {
	res := make([]*GetReleaseResponse, 0, len(list.Releases))
	for _, r := range list.Releases {
		response, err := toGetReleaseResponse(r)
		if err != nil {
			continue
		}
		res = append(res, response)
	}

	return res
}

func GetRequestedChart(rlsName, chartName, chartVersion string, chartPackage []byte, env *helmenv.EnvSettings) (requestedChart *chart.Chart, err error) {
	// If the request has a chart package sent by the user we install that
	if chartPackage != nil && len(chartPackage) != 0 {
		requestedChart, err = chartutil.LoadArchive(bytes.NewReader(chartPackage))
	} else {
		klog.V(4).Infof("Deploying chart=%q, version=%q rlsName=%q", chartName, chartVersion, rlsName)
		var downloadedChartPath string
		downloadedChartPath, err = DownloadChartFromRepo(chartName, chartVersion, env)
		if err != nil {
			return nil, errors.Wrap(err, "error downloading chart")
		}

		requestedChart, err = chartutil.Load(downloadedChartPath)
		klog.V(4).Infof("downloadedChartPath:%s", downloadedChartPath)
	}

	if err != nil {
		return nil, errors.Wrap(err, "error loading chart")
	}

	if req, err := chartutil.LoadRequirements(requestedChart); err == nil {
		if err := checkDependencies(requestedChart, req); err != nil {
			return nil, errors.Wrap(err, "error checking chart dependencies")
		}
	} else if err != chartutil.ErrRequirementsNotFound {
		return nil, errors.Wrap(err, "cannot load requirements")
	}

	return requestedChart, err
}

// UpgradeRelease upgrades a Helm release
func UpgradeRelease(rlsName, chartName, chartVersion string, chartPackage []byte, hClient *Client, overrideValue []byte, reuseValues bool) (*rls.UpdateReleaseResponse, error) {
	chartRequested, err := GetRequestedChart(rlsName, chartName, chartVersion, chartPackage, hClient.Env)
	if err != nil {
		return nil, fmt.Errorf("loading chart has an error: %v", err)
	}

	upgradeRes, err := hClient.UpdateReleaseFromChart(
		rlsName,
		chartRequested,
		helm.UpdateValueOverrides(overrideValue),
		helm.UpgradeDryRun(false),
		// helm.ResetValues(true),
		helm.ReuseValues(reuseValues),
	)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to upgrade a release")
	}

	return upgradeRes, nil
}

func UpgradeReleaseWarp(rlsName string, chartPackage []byte, overrideValue []byte, hClient *Client) (*rls.UpdateReleaseResponse, error) {
	return UpgradeRelease(rlsName, "", "", chartPackage, hClient, overrideValue, true)
}

// CreateRelease creates a Helm release in chosen namespace
func CreateRelease(rlsName, chartName, chartVersion string, chartPackage []byte,
	hClient *Client, namespace string, overrideOpts ...helm.InstallOption) (*rls.InstallReleaseResponse, error) {
	chartRequested, err := GetRequestedChart(rlsName, chartName, chartVersion, chartPackage, hClient.Env)
	if err != nil {
		return nil, fmt.Errorf("error loading chart: %v", err)
	}

	if len(strings.TrimSpace(rlsName)) == 0 {
		rlsName, _ = generateName("")
	}

	if namespace == "" {
		klog.Warningf("rlsName: %s namespace was not set failing back to default", rlsName)
		namespace = DefaultNamespace
	}

	basicOptions := []helm.InstallOption{
		helm.ReleaseName(rlsName),
		helm.InstallDryRun(false),
	}
	installOptions := append(DefaultInstallOptions, basicOptions...)
	installOptions = append(installOptions, overrideOpts...)

	installRes, err := hClient.InstallReleaseFromChart(
		chartRequested,
		namespace,
		installOptions...,
	)
	if err != nil {
		return nil, errors.Wrapf(err, "Error deploying chart")
	}
	return installRes, nil
}

func CreateReleaseWarp(rlsName string, chartPackage []byte, hClient *Client, namespace string, overrideOpts ...helm.InstallOption) (*rls.InstallReleaseResponse, error) {
	return CreateRelease(rlsName, "", "", chartPackage, hClient, namespace, overrideOpts...)
}

// DeleteRelease deletes a Helm release
func DeleteRelease(rlsName string, hClient *Client) error {
	opts := []helm.DeleteOption{
		helm.DeletePurge(true),
	}
	_, err := hClient.DeleteRelease(rlsName, opts...)
	if err != nil {
		return err
	}
	return nil
}

// Create or update a release, creating it if release parameter is nil, otherwise, updating it.
func ApplyRelease(rlsName, chartUrlName, specChartVersion string, chartPackage []byte, hClient *Client,
	namespace string, runningRls *hapirelease.Release, vaByte []byte) (*hapirelease.Release, error) {
	var (
		appliedRls *hapirelease.Release
		rlsErr     error
	)

	if specChartVersion == "" && len(chartPackage) > 0 {
		c, err := chartutil.LoadArchive(bytes.NewReader(chartPackage))
		if err == nil {
			specChartVersion = c.GetMetadata().GetVersion()
		}
	}

	// If the release need to apply is nil, we create this release directly.
	if runningRls == nil {
		rep, err := CreateRelease(rlsName, chartUrlName, specChartVersion, chartPackage, hClient, namespace, helm.ValueOverrides(vaByte))
		if err == nil && rep != nil {
			appliedRls = rep.GetRelease()
			klog.V(4).Infof("Release[%s] has been installed successfully, current version: %d", rlsName, appliedRls.GetVersion())
		}
		rlsErr = err
	} else {
		// If the release need to apply has been passed here, it is necessary to compare it with the running release.
		var isDifferent int

		if specChartVersion != "" {
			runningVersion := runningRls.GetChart().GetMetadata().GetVersion()
			if strings.Compare(specChartVersion, runningVersion) != 0 {
				klog.V(3).Infof("Release[%s] chart version will changed, runningVersion %s => specChartVersion %s", rlsName, runningVersion, specChartVersion)
				isDifferent++
			}
		}

		if isDifferent <= 0 {
			runningRaw := runningRls.GetConfig().GetRaw()
			if len(runningRaw) < 10 && len(vaByte) < 10 {
				klog.V(4).Infof("Release[%s] the length of running raw and spec raw less than 10.", rlsName)
				return nil, nil
			}

			isEquivalent := equality.Semantic.DeepEqual(string(vaByte), runningRaw)
			if isEquivalent {
				klog.V(5).Infof("Release[%s]'s running raw and spec raw not changed, ignore", rlsName)
				return nil, nil
			} else {
				isDifferent++
			}
		}

		// if the running release differ with the spec one, we update it with the spec one.
		if isDifferent > 0 {
			rep, err := UpgradeRelease(rlsName, chartUrlName, specChartVersion, chartPackage, hClient, vaByte, false)
			if err == nil && rep != nil {
				appliedRls = rep.GetRelease()
				klog.V(4).Infof("Release[%s] has been upgraded successfully, current version: %d", rlsName, appliedRls.GetVersion())
			}
			rlsErr = err
		}
	}

	return appliedRls, rlsErr
}

// GetReleaseK8sResources returns K8s resources of a helm release
func GetReleaseK8sResources(rlsName string, hClient *Client, resourceTypes []string) ([]ReleaseResource, error) {
	releaseContent, err := hClient.ReleaseContent(rlsName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, &ReleaseNotFoundError{HelmError: err}
		}
		return nil, err
	}

	return ParseReleaseManifest(releaseContent.Release.Manifest, resourceTypes)
}

func ParseReleaseManifest(manifest string, resourceTypes []string) ([]ReleaseResource, error) {
	objects := strings.Split(manifest, "---")
	decode := scheme.Codecs.UniversalDeserializer().Decode
	releases := make([]ReleaseResource, 0)

	for _, object := range objects {
		if !strings.Contains(object, "apiVersion") {
			continue
		}

		obj, _, err := decode([]byte(object), nil, nil)
		if err != nil {
			klog.Warningf("Error while decoding YAML object. Err was: %s", err)
			continue
		}

		klog.V(3).Infof("version: %v/%v kind: %v", obj.GetObjectKind().GroupVersionKind().Group,
			obj.GetObjectKind().GroupVersionKind().Version, obj.GetObjectKind().GroupVersionKind().Kind)
		selectResource := false
		if len(resourceTypes) == 0 {
			selectResource = true
		} else {
			for _, resourceType := range resourceTypes {
				if strings.ToLower(resourceType) == strings.ToLower(obj.GetObjectKind().GroupVersionKind().Kind) {
					selectResource = true
				}
			}
		}

		if selectResource {
			releases = append(releases, ReleaseResource{
				Name: reflect.ValueOf(obj).Elem().FieldByName("Name").String(),
				Kind: reflect.ValueOf(obj).Elem().FieldByName("Kind").String(),
			})
		}
	}

	return releases, nil
}

// GetRelease returns the details of a helm release
func GetRelease(rlsName string, hClient *Client) (*GetReleaseResponse, error) {
	return GetReleaseByVersion(rlsName, hClient, 0)
}

func toGetReleaseResponse(rls *hapirelease.Release) (*GetReleaseResponse, error) {
	createdAt := time.Unix(rls.GetInfo().GetFirstDeployed().GetSeconds(), 0)
	updatedAt := time.Unix(rls.GetInfo().GetLastDeployed().GetSeconds(), 0)
	chartName := GetVersionedChartName(rls.GetChart().GetMetadata().GetName(), rls.GetChart().GetMetadata().GetVersion())
	notes := base64.StdEncoding.EncodeToString([]byte(rls.GetInfo().GetStatus().GetNotes()))
	cfg, err := chartutil.CoalesceValues(rls.GetChart(), rls.GetConfig())
	if err != nil {
		klog.Errorf("Retrieving release values failed: %s", err.Error())
		return nil, err
	}

	manifest, err := ParseReleaseManifest(rls.Manifest, []string{})
	values := cfg.AsMap()
	return &GetReleaseResponse{
		ReleaseName:  rls.GetName(),
		Namespace:    rls.GetNamespace(),
		Version:      rls.GetVersion(),
		Description:  rls.GetInfo().GetDescription(),
		Status:       rls.GetInfo().GetStatus().GetCode().String(),
		Notes:        notes,
		CreatedAt:    createdAt,
		Updated:      updatedAt,
		Chart:        chartName,
		ChartName:    rls.GetChart().GetMetadata().GetName(),
		ChartVersion: rls.GetChart().GetMetadata().GetVersion(),
		Values:       values,
		Manifest:     manifest,
	}, nil
}

// GetVersionedChartName returns chart name enriched with version number
func GetVersionedChartName(name, version string) string {
	return fmt.Sprintf("%s-%s", name, version)
}

// GetReleaseByVersion returns the details of a helm release version
func GetReleaseByVersion(rlsName string, hClient *Client, version int32) (*GetReleaseResponse, error) {
	rlsInfo, err := hClient.ReleaseContent(rlsName, helm.ContentReleaseVersion(version))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, &ReleaseNotFoundError{HelmError: err}
		}
		return nil, err
	}

	return toGetReleaseResponse(rlsInfo.GetRelease())
}

// GetReleaseStatus retrieves the status of the passed in release name.
// returns with an error if the release is not found or another error occurs
// in case of error the status is filled with information to classify the error cause
func GetReleaseStatus(rlsName string, hClient *Client) (int32, error) {
	releaseStatusResponse, err := hClient.ReleaseStatus(rlsName)
	if err != nil {
		// the release cannot be found
		return http.StatusNotFound, errors.Wrap(err, "couldn't get the release status")
	}

	return int32(releaseStatusResponse.Info.Status.GetCode()), nil
}

func generateName(nameTemplate string) (string, error) {
	t, err := template.New("name-template").Funcs(sprig.TxtFuncMap()).Parse(nameTemplate)
	if err != nil {
		return "", err
	}
	var b bytes.Buffer
	err = t.Execute(&b, nil)
	if err != nil {
		return "", err
	}
	return b.String(), nil
}

func checkDependencies(ch *chart.Chart, reqs *chartutil.Requirements) error {
	missing := []string{}

	deps := ch.GetDependencies()
	for _, r := range reqs.Dependencies {
		found := false
		for _, d := range deps {
			if d.Metadata.Name == r.Name {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, r.Name)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("found in requirements.yaml, but missing in charts/ directory: %s", strings.Join(missing, ", "))
	}
	return nil
}

func mergeValues(dest map[string]interface{}, src map[string]interface{}) map[string]interface{} {
	for k, v := range src {
		// If the key doesn't exist already, then just set the key to that value
		if _, exists := dest[k]; !exists {
			dest[k] = v
			continue
		}
		nextMap, ok := v.(map[string]interface{})
		// If it isn't another map, overwrite the value
		if !ok {
			dest[k] = v
			continue
		}
		// If the key doesn't exist already, then just set the key to that value
		if _, exists := dest[k]; !exists {
			dest[k] = nextMap
			continue
		}
		// Edge case: If the key exists in the destination, but isn't a map
		destMap, isMap := dest[k].(map[string]interface{})
		// If the source map has a map for this key, prefer it
		if !isMap {
			dest[k] = v
			continue
		}
		// If we got to this point, it is a map in both, so merge them
		dest[k] = mergeValues(destMap, nextMap)
	}
	return dest
}
