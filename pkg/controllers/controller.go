package controller

import (
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/advdeployment"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/appset"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/cluster"
	pkgmanager "gitlab.dmall.com/arch/sym-admin/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager) error

// AddToManagerWithCMFuncs is a list of functions to add all Controllers with remote clusters manager to the Manager
var AddToManagerWithCMFuncs []func(manager.Manager, *pkgmanager.DksManager) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager, dksMgr *pkgmanager.DksManager) error {
	if dksMgr != nil {
		if dksMgr.Opt.MasterEnabled {
			AddToManagerWithCMFuncs = append(AddToManagerWithCMFuncs, appset.Add)
		}

		if dksMgr.Opt.WorkerEnabled {
			AddToManagerWithCMFuncs = append(AddToManagerWithCMFuncs, advdeployment.Add)
		}

		if dksMgr.Opt.ClusterEnabled {
			AddToManagerWithCMFuncs = append(AddToManagerWithCMFuncs, cluster.Add)
		}
	}

	for _, f := range AddToManagerFuncs {
		if err := f(m); err != nil {
			return err
		}
	}

	for _, f := range AddToManagerWithCMFuncs {
		if err := f(m, dksMgr); err != nil {
			return err
		}
	}
	return nil
}
