package v2repo

import (
	"time"

	helmv2 "gitlab.dmall.com/arch/sym-admin/pkg/helm/v2"
	"k8s.io/apimachinery/pkg/util/wait"
	helmenv "k8s.io/helm/pkg/helm/environment"
	"k8s.io/klog"
)

const (
	defaultInterval = 60 * 30
)

// HelmIndexSyncer sync helm repo index repeatedly
type HelmIndexSyncer struct {
	Helmv2env *helmenv.EnvSettings

	// interval is the interval of the sync process
	Interval int
}

func NewDefaultHelmIndexSyncer(helmEnv *helmenv.EnvSettings) *HelmIndexSyncer {
	return &HelmIndexSyncer{
		Helmv2env: helmEnv,
		Interval:  defaultInterval,
	}
}

func (h *HelmIndexSyncer) Start(stop <-chan struct{}) error {
	wait.Until(func() {
		klog.V(4).Infof("update helm repo index, time: %v", time.Now())
		entrys, err := helmv2.ReposGet(h.Helmv2env)
		if err != nil {
			klog.Errorf("get all repo err: %+v", err)
			return
		}

		for _, e := range entrys {
			err := helmv2.ReposUpdate(h.Helmv2env, e.Name)
			if err != nil {
				klog.Errorf("updatef repo: %s err: %+v", e.Name, err)
				return
			}

			// lists, err := pkgHelm.ChartsGet(i.Env, "", e.Name, "", "")
			// if err == nil {
			// 	klog.Infof("charts len:%d", len(lists[0].Charts))
			// }
		}
	}, time.Second*time.Duration(h.Interval), stop)
	return nil
}
