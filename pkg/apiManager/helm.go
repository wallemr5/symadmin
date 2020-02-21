package apiManager

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	helmv2 "gitlab.dmall.com/arch/sym-admin/pkg/helm/v2"
	k8smanager "gitlab.dmall.com/arch/sym-admin/pkg/k8s/manager"
	"gitlab.dmall.com/arch/sym-admin/pkg/labels"
	rls "k8s.io/helm/pkg/proto/hapi/services"
	"k8s.io/klog"
)

// GetHelmReleases ...
func (m *APIManager) GetHelmReleases(c *gin.Context) {
	clusterName := c.Param("name")
	appName := c.Param("appName")
	group := c.DefaultQuery("group", "")
	clusters := m.K8sMgr.GetAll(clusterName)

	blue := make([]*model.HelmRelease, 0)
	green := make([]*model.HelmRelease, 0)
	canary := make([]*model.HelmRelease, 0)
	for _, cluster := range clusters {
		response, err := getHelmRelease(cluster, appName, group, "")
		if err != nil {
			AbortHTTPError(c, GetHelmReleasesError, "", err)
			return
		}
		for _, release := range response.GetReleases() {
			item := &model.HelmRelease{
				Cluster:           cluster.GetName(),
				Group:             getGroupFromHelmRelease(release.GetName()),
				Name:              release.GetName(),
				Version:           release.Chart.GetMetadata().GetVersion(),
				Description:       release.Chart.GetMetadata().GetDescription(),
				Status:            release.GetInfo().GetStatus().GetCode().String(),
				FirstDeployedDate: time.Unix(release.Info.FirstDeployed.GetSeconds(), 0).Format("2006-01-02 15:04:05"),
				LastDeployedDate:  time.Unix(release.Info.LastDeployed.GetSeconds(), 0).Format("2006-01-02 15:04:05"),
			}
			switch item.Group {
			case "blue":
				blue = append(blue, item)
			case "green":
				green = append(green, item)
			case "canary":
				canary = append(canary, item)
			}
		}
	}
	c.IndentedJSON(http.StatusOK, gin.H{
		"success": true,
		"message": nil,
		"resultMap": gin.H{
			"greenReleases":  green,
			"blueReleases":   blue,
			"canaryReleases": canary,
		},
	})
}

// GetHelmReleaseInfo ...
func (m *APIManager) GetHelmReleaseInfo(c *gin.Context) {
	clusterName := c.Param("name")
	releaseName := c.Param("releaseName")
	cluster, err := m.K8sMgr.Get(clusterName)
	if err != nil {
		AbortHTTPError(c, GetClusterError, "", err)
		return
	}

	response, err := getHelmRelease(cluster, "", "", releaseName)
	if err != nil {
		AbortHTTPError(c, GetHelmReleasesError, "", err)
		return
	}
	result := &model.HelmWholeRelease{}
	for _, release := range response.Releases {
		if release.GetName() == releaseName {
			result.Name = release.GetName()
			result.Namespace = release.GetNamespace()
			result.Version = release.GetVersion()
			result.Manifest = release.GetManifest()
			result.Info = &model.Info{
				Status:        &model.Status{Code: release.GetInfo().GetStatus().GetCode().String()},
				Description:   release.GetInfo().GetDescription(),
				FirstDeployed: release.GetInfo().GetFirstDeployed().String(),
				LastDeployed:  release.GetInfo().GetLastDeployed().String(),
			}
			result.Chart = &model.Chart{
				Metadata: &model.Metadata{
					Name:        release.GetChart().GetMetadata().GetName(),
					Description: release.GetChart().GetMetadata().GetDescription(),
					Version:     release.GetChart().GetMetadata().GetVersion(),
					APIVersion:  release.GetChart().GetMetadata().GetApiVersion(),
				},
				Value: &model.Value{Raw: release.GetChart().GetValues().GetRaw()},
			}
			var templates = make([]*model.Template, 0)
			for _, template := range release.Chart.Templates {
				templates = append(templates, &model.Template{Name: template.Name, Data: template.Data})
			}
			result.Chart.Templates = templates
			result.Config = &model.Config{Raw: release.GetConfig().GetRaw()}
		}
	}
	c.IndentedJSON(http.StatusOK, result)
}

func getHelmRelease(cluster *k8smanager.Cluster, appName, group, releaseName string) (*rls.ListReleasesResponse, error) {
	hClient, err := helmv2.NewClientFromConfig(cluster.RestConfig, cluster.KubeCli, "")
	if err != nil {
		klog.Errorf("Initializing a new helm clinet has an error: %+v", err)
		return nil, err
	}
	defer hClient.Close()

	var filter string
	if releaseName != "" {
		filter = releaseName
	} else {
		filter, err = labels.MakeHelmReleaseFilterWithGroup(appName, group)
		if err != nil {
			return nil, err
		}
	}

	response, err := helmv2.ListReleases(filter, hClient)
	if err != nil || response == nil {
		klog.Errorf("Can not find release[%s] before deleting it", appName)
		return nil, err
	}
	return response, nil
}

func getGroupFromHelmRelease(name string) model.GroupEnum {
	switch {
	case strings.Contains(name, "blue"):
		return model.BlueGroup
	case strings.Contains(name, "green"):
		return model.GreenGroup
	case strings.Contains(name, "canary"):
		return model.CanaryGroup
	case strings.Contains(name, "svc"):
		return model.SvcGroup
	default:
		return model.Unkonwn
	}
}
