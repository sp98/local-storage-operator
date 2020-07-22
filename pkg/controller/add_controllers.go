package controller

import (
	"github.com/openshift/local-storage-operator/pkg/controller/localvolume"
	"github.com/openshift/local-storage-operator/pkg/controller/localvolumediscovery"
	"github.com/openshift/local-storage-operator/pkg/controller/localvolumeset"
	"github.com/openshift/local-storage-operator/pkg/controller/nodedaemon"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs,
		localvolume.Add,
		localvolumediscovery.Add,
		localvolumeset.AddLocalVolumeSetReconciler,
		nodedaemon.AddDaemonReconciler,
	)
}
