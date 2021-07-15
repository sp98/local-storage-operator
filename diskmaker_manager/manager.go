package main

import (
	"flag"

	localv1 "github.com/openshift/local-storage-operator/api/v1"
	localv1alpha1 "github.com/openshift/local-storage-operator/api/v1alpha1"
	"github.com/openshift/local-storage-operator/common"
	diskmakerControllerDeleter "github.com/openshift/local-storage-operator/diskmaker/controllers/deleter"
	diskmakerControllerLv "github.com/openshift/local-storage-operator/diskmaker/controllers/lv"
	diskmakerControllerLvSet "github.com/openshift/local-storage-operator/diskmaker/controllers/lvset"
	"github.com/openshift/local-storage-operator/localmetrics"
	"github.com/pkg/errors"
	"github.com/prometheus/common/log"
	"github.com/spf13/cobra"
	apiruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	provCache "sigs.k8s.io/sig-storage-local-static-provisioner/pkg/cache"
	provDeleter "sigs.k8s.io/sig-storage-local-static-provisioner/pkg/deleter"
)

var (
	scheme   = apiruntime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(localv1.AddToScheme(scheme))
	utilruntime.Must(localv1alpha1.AddToScheme(scheme))
}

func startManager(cmd *cobra.Command, args []string) error {
	klogFlags := flag.NewFlagSet("local-storage-diskmaker", flag.ExitOnError)
	klog.InitFlags(klogFlags)
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Set("alsologtostderr", "true")
	flag.Parse()
	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	printVersion()

	namespace, err := common.GetWatchNamespace()
	if err != nil {
		log.Error(err, "Failed to get watch namespace")
		return err
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Namespace:          namespace,
		Scheme:             scheme,
		MetricsBindAddress: "0",
		LeaderElection:     false,
	})
	if err != nil {
		setupLog.Error(err, "")
		return err
	}

	if err = (&diskmakerControllerLv.LocalVolumeReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr, &provDeleter.CleanupStatusTracker{ProcTable: provDeleter.NewProcTable()}, provCache.NewVolumeCache()); err != nil {
		setupLog.Error(err, "unable to create diskmaker controller", "controller", "LocalVolume")
		return err
	}

	if err = (&diskmakerControllerLvSet.LocalVolumeSetReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr, &provDeleter.CleanupStatusTracker{ProcTable: provDeleter.NewProcTable()}, provCache.NewVolumeCache()); err != nil {
		setupLog.Error(err, "unable to create diskmaker controller", "controller", "LocalVolumeSet")
		return err
	}

	if err = (&diskmakerControllerDeleter.DeleteReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr, &provDeleter.CleanupStatusTracker{ProcTable: provDeleter.NewProcTable()}, provCache.NewVolumeCache()); err != nil {
		setupLog.Error(err, "unable to create diskmaker controller", "controller", "Deleter")
		return err
	}

	// start local server to emit custom metrics
	// TODO add LocalVolume and LocalVolumeSet metric collectors
	err = localmetrics.NewConfigBuilder().Build()
	if err != nil {
		return errors.Wrap(err, "failed to configure local metrics")
	}

	// Start the Cmd
	stopChan := signals.SetupSignalHandler()
	if err := mgr.Start(stopChan); err != nil {
		log.Error(err, "manager exited non-zero")
		return err
	}
	return nil
}
