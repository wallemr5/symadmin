package object

import (
	"bytes"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/renderutil"
	"k8s.io/helm/pkg/timeconv"
)

type ReleaseOptions chartutil.ReleaseOptions

func GetDefaultValues(fs http.FileSystem) ([]byte, error) {
	file, err := fs.Open(chartutil.ValuesfileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(file)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read default values")
	}

	return buf.Bytes(), nil
}

func Render(fs http.FileSystem, values string, releaseOptions ReleaseOptions, chartName string) (K8sObjects, error) {
	chrtConfig := &chart.Config{
		Raw:    values,
		Values: map[string]*chart.Value{},
	}

	files, err := getFiles(fs)
	if err != nil {
		return nil, err
	}

	// Create chart and render templates
	chrt, err := chartutil.LoadFiles(files)
	if err != nil {
		return nil, err
	}

	renderOpts := renderutil.Options{
		ReleaseOptions: chartutil.ReleaseOptions{
			Name:      releaseOptions.Name,
			IsInstall: true,
			IsUpgrade: false,
			Time:      timeconv.Now(),
			Namespace: releaseOptions.Namespace,
		},
		KubeVersion: "",
	}

	renderedTemplates, err := renderutil.Render(chrt, chrtConfig, renderOpts)
	if err != nil {
		return nil, err
	}

	// Merge templates and inject
	var buf bytes.Buffer
	for _, tmpl := range files {
		if !strings.HasSuffix(tmpl.Name, "yaml") && !strings.HasSuffix(tmpl.Name, "yml") && !strings.HasSuffix(tmpl.Name, "tpl") {
			continue
		}
		t := path.Join(chartName, tmpl.Name)
		if _, err := buf.WriteString(renderedTemplates[t]); err != nil {
			return nil, err
		}
		buf.WriteString("\n---\n")
	}

	objects, err := ParseK8sObjectsFromYAMLManifest(buf.String())
	if err != nil {
		return nil, err
	}

	return objects, nil
}

func getFiles(fs http.FileSystem) ([]*chartutil.BufferedFile, error) {
	files := []*chartutil.BufferedFile{
		{
			Name: chartutil.ChartfileName,
		},
	}

	// if the Helm chart templates use some resource files (like dashboards), those should be put under resources
	for _, dirName := range []string{"resources", chartutil.TemplatesDir} {
		dir, err := fs.Open(dirName)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}
		} else {
			dirFiles, err := dir.Readdir(-1)
			if err != nil {
				return nil, err
			}

			for _, file := range dirFiles {
				filename := file.Name()
				if strings.HasSuffix(filename, "yaml") || strings.HasSuffix(filename, "yml") || strings.HasSuffix(filename, "tpl") || strings.HasSuffix(filename, "json") {
					files = append(files, &chartutil.BufferedFile{
						Name: dirName + "/" + filename,
					})
				}
			}
		}
	}

	for _, f := range files {
		data, err := readIntoBytes(fs, f.Name)
		if err != nil {
			return nil, err
		}

		f.Data = data
	}

	return files, nil
}

func readIntoBytes(fs http.FileSystem, filename string) ([]byte, error) {
	file, err := fs.Open(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "could not open file")
	}
	defer file.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(file)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read file")
	}

	return buf.Bytes(), nil
}

func InstallObjectOrder() func(o *K8sObject) int {
	var Order = []string{
		"CustomResourceDefinition",
		"Namespace",
		"ResourceQuota",
		"LimitRange",
		"PodSecurityPolicy",
		"PodDisruptionBudget",
		"Secret",
		"ConfigMap",
		"StorageClass",
		"PersistentVolume",
		"PersistentVolumeClaim",
		"ServiceAccount",
		"ClusterRole",
		"ClusterRoleList",
		"ClusterRoleBinding",
		"ClusterRoleBindingList",
		"Role",
		"RoleList",
		"RoleBinding",
		"RoleBindingList",
		"Service",
		"DaemonSet",
		"Pod",
		"ReplicationController",
		"ReplicaSet",
		"Deployment",
		"HorizontalPodAutoscaler",
		"StatefulSet",
		"Job",
		"CronJob",
		"Ingress",
		"APIService",
	}

	order := make(map[string]int, len(Order))
	for i, kind := range Order {
		order[kind] = i
	}

	return func(o *K8sObject) int {
		if nr, ok := order[o.Kind]; ok {
			return nr
		}
		return 1000
	}
}

func UninstallObjectOrder() func(o *K8sObject) int {
	var Order = []string{
		"APIService",
		"Ingress",
		"Service",
		"CronJob",
		"Job",
		"StatefulSet",
		"HorizontalPodAutoscaler",
		"Deployment",
		"ReplicaSet",
		"ReplicationController",
		"Pod",
		"DaemonSet",
		"RoleBindingList",
		"RoleBinding",
		"RoleList",
		"Role",
		"ClusterRoleBindingList",
		"ClusterRoleBinding",
		"ClusterRoleList",
		"ClusterRole",
		"ServiceAccount",
		"PersistentVolumeClaim",
		"PersistentVolume",
		"StorageClass",
		"ConfigMap",
		"Secret",
		"PodDisruptionBudget",
		"PodSecurityPolicy",
		"LimitRange",
		"ResourceQuota",
		"Policy",
		"Gateway",
		"VirtualService",
		"DestinationRule",
		"Handler",
		"Instance",
		"Rule",
		"Namespace",
		"CustomResourceDefinition",
	}

	order := make(map[string]int, len(Order))
	for i, kind := range Order {
		order[kind] = i
	}

	return func(o *K8sObject) int {
		if nr, ok := order[o.Kind]; ok {
			return nr
		}
		return 1000
	}
}
