package offlinepod

import (
	"context"

	"fmt"

	pkgmanager "gitlab.dmall.com/arch/sym-admin/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"encoding/json"

	"time"

	"gitlab.dmall.com/arch/sym-admin/pkg/apiManager/model"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
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

	aggrCm := &corev1.ConfigMap{}
	err := c.Client.Get(ctx, types.NamespacedName{
		Namespace: c.ObvNs,
		Name:      pod.AppName,
	}, aggrCm)
	if err != nil {
		if apierrors.IsNotFound(err) {
			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      pod.AppName,
					Labels:    GetConfigMapLabels(),
					Namespace: c.ObvNs,
				},
			}
			err := c.Client.Create(ctx, cm)
			if err != nil {
				klog.Errorf("failed to create ConfigMap, err: %v", err)
				return err
			}

			aggrCm = cm
		} else {
			klog.Errorf("failed to get ConfigMap, err: %v", err)
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
	if cache, ok = c.Cache[pod.AppName]; !ok {
		cache = New(c.MaxOffline, pod.AppName)
		c.Cache[pod.AppName] = cache
	}

	if len(oldRaw) > 0 && cache.Len() == 0 {
		jerr := json.Unmarshal([]byte(oldRaw), &apps)
		if jerr != nil {
			klog.Errorf("failed to Unmarshal err: %v", jerr)
			return jerr
		}

		klog.Infof("name:%s add old cache pod items len: %d", pod.AppName, len(apps))
		for _, p := range apps {
			cache.Add(p)
		}
	}

	cache.Add(pod)
	apps = cache.List()
	appsByte, jerr := json.MarshalIndent(apps, "", "  ")
	if jerr != nil {
		klog.Errorf("failed to Marshal err: %v", jerr)
		return jerr
	}

	appsRaw := string(appsByte)
	if len(oldRaw) > 0 &&
		equality.Semantic.DeepEqual(appsRaw, oldRaw) {
		return nil
	}

	aggrCm.Data[ConfigDataKey] = appsRaw
	uerr := c.Client.Update(ctx, aggrCm)
	if uerr != nil {
		klog.Errorf("failed to create Update, err: %v", uerr)
		return uerr
	}
	return nil
}
