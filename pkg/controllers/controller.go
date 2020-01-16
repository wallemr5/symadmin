package controller

import (
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/advdeployment"
	"gitlab.dmall.com/arch/sym-admin/pkg/controllers/appset"
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
		// for _, c := range cMgr.Manager.AllWorker() {
		// 	cMgr.HealthHandler.AddReadinessCheck(fmt.Sprintf("%s_%s", c.Name, "deploy_cache_sync"),
		// 		func() error {
		// 			if c.DeploymentInformer.Informer().HasSynced() {
		// 				return nil
		// 			}
		// 			return fmt.Errorf("cluster:%s deploy cache not sync", c.Name)
		// 		})
		// 	cMgr.HealthHandler.AddReadinessCheck(fmt.Sprintf("%s_%s", c.Name, "node_cache_sync"),
		// 		func() error {
		// 			if c.NodeInformer.Informer().HasSynced() {
		// 				return nil
		// 			}
		// 			return fmt.Errorf("cluster:%s node cache not sync", c.Name)
		// 		})
		// }
		//
		// cMgr.HealthHandler.AddLivenessCheck("cluster_db_nochanged",
		// 	func() error {
		// 		if !cMgr.Manager.ClusterDb.IsChanged() {
		// 			return nil
		// 		}
		// 		return fmt.Errorf("cluster is changed")
		// 	})

		if dksMgr.Opt.MasterEnabled {
			AddToManagerWithCMFuncs = append(AddToManagerWithCMFuncs, appset.Add)
		}

		if dksMgr.Opt.WorkerEnabled {
			AddToManagerWithCMFuncs = append(AddToManagerWithCMFuncs, advdeployment.Add)
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
