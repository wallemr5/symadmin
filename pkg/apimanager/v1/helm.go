package v1

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gitlab.dmall.com/arch/sym-admin/pkg/apimanager/model"
	"gitlab.dmall.com/arch/sym-admin/pkg/helm/object"
	k8sclient "gitlab.dmall.com/arch/sym-admin/pkg/k8s/client"
	"gitlab.dmall.com/arch/sym-admin/pkg/resources"
	"gitlab.dmall.com/arch/sym-admin/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/util/retry"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	objTemp = `
---
# Source: api/templates/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: test-api
  namespace: sym-admin
  labels:
    app.kubernetes.io/name: api
    helm.sh/chart: api-1.0.17
    app.kubernetes.io/instance: test-api
    app.kubernetes.io/managed-by: Helm
spec:
  type: ClusterIP
  ports:
    - name: http
      port: 8080
      targetPort: 8080
      protocol: TCP
  selector:
    app.kubernetes.io/name: api
    app.kubernetes.io/instance: test-api
---
# Source: api/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-api
  namespace: sym-admin
  labels:
    app.kubernetes.io/name: api
    helm.sh/chart: api-1.0.17
    app.kubernetes.io/instance: test-api
    app.kubernetes.io/managed-by: Helm
spec:
  replicas: 4
  selector:
    matchLabels:
      app.kubernetes.io/name: api
      app.kubernetes.io/instance: test-api
  template:
    metadata:
      labels:
        app.kubernetes.io/name: api
        app.kubernetes.io/instance: test-api
    spec:
      containers:
        - name: api
          image: "symcn.tencentcloudcr.com/symcn/sym-admin-api:v1.0.10"
          imagePullPolicy: IfNotPresent
          args:
          - "api"
          - "-v"
          - "4"
          ports:
            - name: http
              containerPort: 8080
              protocol: TCP
          livenessProbe:
            httpGet:
              path: /ready
              port: http
            initialDelaySeconds: 5
            periodSeconds: 30
          readinessProbe:
            httpGet:
              path: /ready
              port: http
            initialDelaySeconds: 5
            periodSeconds: 30
          resources:
            limits:
              cpu: 3
              memory: 1Gi
            requests:
              cpu: 1
              memory: 256Mi
      imagePullSecrets:
        - name: tencenthubkey
      serviceAccountName: api
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app.kubernetes.io/name
                  operator: In
                  values:
                  - api
              topologyKey: kubernetes.io/hostname
            weight: 100
      hostAliases:
        - hostnames:
          - cls-89a4hpb3.ccs.tencent-cloud.com
          ip: 10.13.135.251
        - hostnames:
          - cls-cm580t93.ccs.tencent-cloud.com
          ip: 10.13.134.9
        - hostnames:
          - cls-0snem5sv.ccs.tencent-cloud.com
          ip: 10.13.133.7
        - hostnames:
          - cls-7xq1bq9f.ccs.tencent-cloud.com
          ip: 10.13.135.12
        - hostnames:
          - cls-otdyiqyb.ccs.tencent-cloud.com
          ip: 10.16.247.78
        - hostnames:
          - cls-h5f02nmb.ccs.tencent-cloud.com
          ip: 10.16.247.11
        - hostnames:
          - cls-3yclxq8t.ccs.tencent-cloud.com
          ip: 10.16.113.12
        - hostnames:
          - cls-0snem5sv.ccs.tencent-cloud.com
          ip: 10.13.133.9
        - hostnames:
          - cls-278pwqet.ccs.tencent-cloud.com
          ip: 10.16.247.131
        - hostnames:
          - cls-97rlivuj.ccs.tencent-cloud.com
          ip: 10.16.113.81
        - hostnames:
          - cls-azg4i2et.ccs.tencent-cloud.com
          ip: 10.13.133.134
        - hostnames:
          - cls-glojus0v.ccs.tencent-cloud.com
          ip: 10.16.70.8
        - hostnames:
          - cls-2ylraskd.ccs.tencent-cloud.com
          ip: 10.248.227.7
        - hostnames:
          - cls-ehx4vson.ccs.tencent-cloud.com
          ip: 10.248.227.74
        - hostnames:
          - cls-0doi9yrf.ccs.tencent-cloud.com
          ip: 10.248.224.193
        - hostnames:
          - chartmuseum.dmall.com
          ip: 10.13.135.250
---
# Source: api/templates/ingress.yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: test-api
  namespace: sym-admin
  labels:
    app.kubernetes.io/name: api
    helm.sh/chart: api-1.0.17
    app.kubernetes.io/instance: test-api
    app.kubernetes.io/managed-by: Helm
  annotations:
    kubernetes.io/ingress.class: traefik
spec:
  rules:
    - host: "devapi.sym.dmall.com"
      http:
        paths:
          - path: /
            backend:
              serviceName: test-api
              servicePort: http
`
)

// GetHelmReleases ...
func (m *Manager) GetHelmReleases(c *gin.Context) {
	// clusterName := c.Param("name")
	// appName := c.Param("appName")
	// group := c.DefaultQuery("group", "")
	// clusters := m.K8sMgr.GetAll(clusterName)
	// zone := c.DefaultQuery("symZone", "")

	c.IndentedJSON(http.StatusOK, gin.H{
		"success": true,
		"message": nil,
	})
}

// GetHelmReleaseInfo ...
func (m *Manager) GetHelmReleaseInfo(c *gin.Context) {
	// clusterName := c.Param("name")
	// releaseName := c.Param("releaseName")
	// cluster, err := m.K8sMgr.Get(clusterName)
	// zone := c.DefaultQuery("symZone", "")
	// if err != nil {
	// 	c.IndentedJSON(http.StatusBadRequest, gin.H{
	// 		"success":   false,
	// 		"message":   err.Error(),
	// 		"resultMap": nil,
	// 	})
	// 	return
	// }

	c.IndentedJSON(http.StatusOK, gin.H{
		"success": true,
		"message": nil,
	})
}

func lintK8sObj(c client.Client, objs object.K8sObjects) (string, string, bool, error) {
	isSuccess := true
	message := "success"
	ctx := context.TODO()
	var buf bytes.Buffer

	// avoid conflict
	ns := "sym-admin"
	s := json.NewSerializerWithOptions(json.DefaultMetaFactory,
		k8sclient.GetScheme(), k8sclient.GetScheme(), json.SerializerOptions{Yaml: true})
	for _, obj := range objs {
		valueStr := obj.YAMLDebugString()
		buf.WriteString("\n---\n")
		buf.WriteString(valueStr)
		orgObj, _, err := s.Decode(utils.String2bytes(valueStr), nil, nil)
		if err != nil {
			klog.Errorf("%s/%sfailed to parse yaml to k8s object err: %+v, yml: \n%s", obj.Kind, obj.Name, err, valueStr)
			return "", "", false, err
			// message = fmt.Sprintf("%s/%s failed to parse yaml to k8s object err: %+v", obj.Kind, obj.Name, err)
			// isSuccess = false
			// continue
		}

		isKnown := true
		convertObj := orgObj.DeepCopyObject()
		switch convertObj.(type) {
		case *appsv1.Deployment:
			deploy := convertObj.(*appsv1.Deployment)
			deploy.Spec.Replicas = utils.IntPointer(0)
			deploy.Namespace = ns
		case *appsv1.StatefulSet:
			sta := convertObj.(*appsv1.StatefulSet)
			sta.Spec.Replicas = utils.IntPointer(0)
			sta.Namespace = ns
		case *corev1.Service:
			svc := convertObj.(*corev1.Service)
			svc.Namespace = ns
		case *v1beta1.Ingress:
			ingress := convertObj.(*v1beta1.Ingress)
			ingress.Namespace = ns
		default:
			msg := fmt.Sprintf("%s/%s unknown kind, yml: \n%s", obj.Kind, obj.Name, valueStr)
			klog.Error(msg)
			return "", "", false, errors.New(msg)
			// isKnown = false
		}

		if !isKnown {
			continue
		}

		_, err = resources.Reconcile(ctx, c, convertObj, resources.Option{DesiredState: resources.DesiredStatePresent})
		if err != nil {
			klog.Errorf("%s/%s failed dry run err: %+v, yml: \n%s", obj.Kind, obj.Name, err, valueStr)
			return "", "", false, err
			// message = fmt.Sprintf("%s/%s failed dry run err: %+v", obj.Kind, obj.Name, err)
			// isSuccess = false
			// continue
		}

		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			time.Sleep(10 * time.Millisecond)
			_, err = resources.Reconcile(ctx, c, convertObj, resources.Option{DesiredState: resources.DesiredStateAbsent})
			if err != nil {
				klog.Errorf("%s/%s failed to delete err: %+v, yml: \n%s", obj.Kind, obj.Name, err, valueStr)
				return err
			}
			return nil
		})
		if err != nil {
			return "", "", false, err
		}
	}

	return buf.String(), message, isSuccess, nil
}

// LintLocalTemplate ...
func (m *Manager) LintLocalTemplate(c *gin.Context) {
	objs, err := object.ParseK8sObjectsFromYAMLManifest(objTemp)
	if err != nil {
		klog.Errorf("failed parse k8s obj err: %+v", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": fmt.Sprintf("failed parse k8s obj err: %+v", err),
		})
		return
	}

	manifest, message, isSuccess, err := lintK8sObj(m.ClustersMgr.MasterClient.GetClient(), objs)
	// c.String(http.StatusOK, manifest)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{
		"success":  isSuccess,
		"message":  message,
		"manifest": manifest,
	})
}

// LintHelmTemplate ...
func (m *Manager) LintHelmTemplate(c *gin.Context) {
	rlsName := c.PostForm("rlsName")
	ns := c.PostForm("namespace")
	overrideValue := c.PostForm("overrideValue")
	chartPkg, header, err := c.Request.FormFile("chart")
	if err != nil {
		klog.Errorf("upload chart file err: %+v", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": fmt.Sprintf("upload chart file err: %+v", err),
		})
		return
	}

	klog.Infof("get upload chart file: %s", header.Filename)
	chartByte, err := ioutil.ReadAll(chartPkg)
	if err != nil {
		klog.Errorf("read chart file err: %+v", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": fmt.Sprintf("read chart file err: %+v", err),
		})
	}

	objs, err := object.RenderTemplate(chartByte, rlsName, ns, overrideValue)
	if err != nil {
		klog.Errorf("lint helm template err: %+v", err)
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": fmt.Sprintf("lint helm template err: %+v", err),
		})
		return
	}

	manifest, message, isSuccess, err := lintK8sObj(m.ClustersMgr.MasterClient.GetClient(), objs)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.IndentedJSON(http.StatusOK, gin.H{
		"success":  isSuccess,
		"message":  message,
		"manifest": manifest,
	})
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
