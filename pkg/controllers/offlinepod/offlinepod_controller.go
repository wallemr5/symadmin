package offlinepod

import (
	"context"
	"fmt"
	"time"

	json "github.com/json-iterator/go"
	"gitlab.dmall.com/arch/sym-admin/pkg/apimanager/model"
	workloadv1beta1 "gitlab.dmall.com/arch/sym-admin/pkg/apis/workload/v1beta1"
	pkgmanager "gitlab.dmall.com/arch/sym-admin/pkg/manager"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func Add(mgr manager.Manager, cMgr *pkgmanager.DksManager) error {
	r, err := NewOfflinepodReconciler(mgr, cMgr)
	if r == nil {
		return fmt.Errorf("NewOfflinepodReconciler err: %v", err)
	}

	err = mgr.Add(r)
	if err != nil {
		klog.Fatal("Can't add runnable for controller")
		return err
	}

	return nil
}

func GetConfigMapLabels() map[string]string {
	return map[string]string{
		"controllerOwner": "offlinePod",
	}
}

func (c *offlinepodImpl) reconciler(ctx context.Context, pod *model.OfflinePod) error {
	if pod == nil {
		return nil
	}

	key := fmt.Sprintf("%s/%s", pod.Namespace, pod.AppName)
	logger := c.Log.WithValues("key", key)
	startTime := time.Now()
	defer func() {
		diffTime := time.Since(startTime)
		var logLevel klog.Level
		if diffTime > 1*time.Second {
			logLevel = 1
		} else if diffTime > 100*time.Millisecond {
			logLevel = 2
		} else {
			logLevel = 4
		}
		klog.V(logLevel).Infof("##### [%s] reconciling is finished. time taken: %v. ", pod.Name, diffTime)
	}()

	as := &workloadv1beta1.AppSet{}
	err := c.Client.Get(ctx, types.NamespacedName{
		Namespace: pod.Namespace,
		Name:      pod.AppName,
	}, as)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Can't find AppSet")
			return nil
		}

		logger.Error(err, "failed to get AppSet")
		return err
	}

	aggrCm := &corev1.ConfigMap{}
	err = c.Client.Get(ctx, types.NamespacedName{
		Namespace: pod.Namespace,
		Name:      pod.AppName,
	}, aggrCm)
	if err != nil {
		if apierrors.IsNotFound(err) {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pod.AppName,
					Labels:    GetConfigMapLabels(),
					Namespace: pod.Namespace,
				},
			}

			err = controllerutil.SetControllerReference(as, cm, c.MasterMgr.GetScheme())
			if err != nil {
				logger.Error(err, "failed to set reference with offline configmap")
				return err
			}

			err := c.Client.Create(ctx, cm)
			if err != nil {
				logger.Error(err, "failed to create offline configmap")
				return err
			}

			aggrCm = cm
		} else {
			logger.Error(err, "failed to get offline configmap")
			return err
		}
	}

	var (
		oldRaw string
		ok     bool
		cache  *Cache
		apps   []*model.OfflinePod
	)

	if aggrCm.Data == nil {
		aggrCm.Data = make(map[string]string)
	}

	oldRaw = aggrCm.Data[ConfigDataKey]
	if cache, ok = c.Cache[key]; !ok {
		maxOffline := c.MaxOffline
		if as.Status.AggrStatus.Desired > c.MaxOffline {
			maxOffline = as.Status.AggrStatus.Desired
			logger.Info("set cache", "max offline", maxOffline)
		}
		cache = New(maxOffline, key, c)
		c.Cache[key] = cache
	}

	if as.Status.AggrStatus.Desired > c.MaxOffline && cache.GetMaxEntries() != as.Status.AggrStatus.Desired {
		cache.SetMaxEntries(as.Status.AggrStatus.Desired)
	}
	if len(oldRaw) > 0 && cache.Len() == 0 {
		jerr := json.ConfigCompatibleWithStandardLibrary.Unmarshal([]byte(oldRaw), &apps)
		if jerr != nil {
			logger.Error(jerr, "failed to Unmarshal offlineList")
			return jerr
		}

		logger.Info("add old cache list", "items", len(apps))

		// latst data add to list front
		for i := len(apps) - 1; i >= 0; i-- {
			cache.Add(apps[i])
		}
	}

	cache.Add(pod)
	apps = cache.List()
	appsByte, jerr := json.ConfigCompatibleWithStandardLibrary.MarshalIndent(apps, "", "  ")
	if jerr != nil {
		logger.Error(jerr, "failed to Marshal offlineList")
		return jerr
	}

	appsRaw := string(appsByte)
	if len(oldRaw) > 0 &&
		equality.Semantic.DeepEqual(appsRaw, oldRaw) {
		return nil
	}

	// re get
	upCm := &corev1.ConfigMap{}
	err = c.Client.Get(ctx, types.NamespacedName{
		Namespace: pod.Namespace,
		Name:      pod.AppName,
	}, upCm)
	if err != nil {
		logger.Error(err, "failed to get offlineList configmap")
		return err
	}
	if upCm.Data == nil {
		upCm.Data = make(map[string]string)
	}
	upCm.Data[ConfigDataKey] = appsRaw
	uerr := c.Client.Update(ctx, upCm)
	if uerr != nil {
		logger.Error(uerr, "failed to update offlineList configmap")
		return uerr
	}

	logger.Info("offline pod update success", "items", len(apps))
	return nil
}
