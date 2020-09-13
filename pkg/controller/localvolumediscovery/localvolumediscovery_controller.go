package localvolumediscovery

import (
	"context"
	"fmt"
	"strings"
	"time"

	operatorv1 "github.com/openshift/api/operator/v1"
	localv1alpha1 "github.com/openshift/local-storage-operator/pkg/apis/local/v1alpha1"
	"github.com/openshift/local-storage-operator/pkg/common"
	"github.com/openshift/local-storage-operator/pkg/controller/nodedaemon"
	"github.com/openshift/local-storage-operator/pkg/diskmaker/discovery"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	log                             = logf.Log.WithName("controller_localvolumediscovery")
	waitForRequeueIfDaemonsNotReady = reconcile.Result{Requeue: true, RequeueAfter: 10 * time.Second}
)

const (
	udevPath           = "/run/udev"
	udevVolName        = "run-udev"
	DiskMakerDiscovery = "diskmaker-discovery"
)

// Add creates a new LocalVolumeDiscovery Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileLocalVolumeDiscovery{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("localvolumediscovery-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource LocalVolumeDiscovery
	err = c.Watch(&source.Kind{Type: &localv1alpha1.LocalVolumeDiscovery{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes in the secondary resource Daemonset and requeue LocalVolumeDiscovery
	err = c.Watch(&source.Kind{Type: &appsv1.DaemonSet{}}, &handler.EnqueueRequestForOwner{OwnerType: &localv1alpha1.LocalVolumeDiscovery{}})
	if err != nil {
		return err
	}

	// watch for changes in the secondary resource Persistence Volume Claim and requeue LocalVolumeDiscovery
	err = c.Watch(&source.Kind{Type: &corev1.PersistentVolume{}}, &handler.EnqueueRequestForObject{})
	return nil

}

// blank assignment to verify that ReconcileLocalVolumeDiscovery implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileLocalVolumeDiscovery{}

// ReconcileLocalVolumeDiscovery reconciles a LocalVolumeDiscovery object
type ReconcileLocalVolumeDiscovery struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a LocalVolumeDiscovery object and makes changes based on the state read
// and what is in the LocalVolumeDiscovery.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileLocalVolumeDiscovery) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling LocalVolumeDiscovery")

	// Fetch the LocalVolumeDiscovery instance
	instance := &localv1alpha1.LocalVolumeDiscovery{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			if strings.Contains(request.Name, "local-pv") {
				instanceList := &localv1alpha1.LocalVolumeDiscoveryList{}
				err := r.client.List(context.TODO(), instanceList)
				if err != nil {
					return reconcile.Result{}, err
				}
				if len(instanceList.Items) > 0 {
					// only one localVolumeDiscovery can be created. So always use the first instance.
					instance = &instanceList.Items[0]
				} else {
					// persitence volume claim was created but no localvolumediscovery is created yet.
					// Return and don't requeue
					return reconcile.Result{}, nil
				}
			} else {
				// Request object not found, could have been deleted after reconcile request.
				// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
				// Return and don't requeue
				return reconcile.Result{}, nil
			}
		} else {
			// Error reading the object - requeue the request.
			return reconcile.Result{}, err
		}
	}

	diskMakerDSMutateFn := getDiskMakerDiscoveryDSMutateFn(request, instance.Spec.Tolerations, getOwnerRefs(instance), instance.Spec.NodeSelector)
	ds, opResult, err := nodedaemon.CreateOrUpdateDaemonset(r.client, diskMakerDSMutateFn)
	if err != nil {
		message := fmt.Sprintf("failed to create discovery daemonset. Error %+v", err)
		err := r.updateDiscoveryStatus(instance, operatorv1.OperatorStatusTypeDegraded, message,
			operatorv1.ConditionFalse, localv1alpha1.DiscoveryFailed)
		if err != nil {
			return reconcile.Result{}, err
		}
	} else if opResult == controllerutil.OperationResultUpdated || opResult == controllerutil.OperationResultCreated {
		reqLogger.Info("daemonset changed", "daemonset.Name", ds.GetName(), "op.Result", opResult)
	}

	desiredDaemons, readyDaemons, err := r.getDaemonSetStatus(instance.Namespace)
	if err != nil {
		reqLogger.Error(err, "failed to get discovery daemonset")
		return reconcile.Result{}, err
	}

	if desiredDaemons == 0 {
		message := "no discovery daemons are scheduled for running"
		err := r.updateDiscoveryStatus(instance, operatorv1.OperatorStatusTypeDegraded, message,
			operatorv1.ConditionFalse, localv1alpha1.DiscoveryFailed)
		if err != nil {
			return reconcile.Result{}, err
		}
		return waitForRequeueIfDaemonsNotReady, fmt.Errorf(message)

	} else if !(desiredDaemons == readyDaemons) {
		message := fmt.Sprintf("running %d out of %d discovery daemons", readyDaemons, desiredDaemons)
		err := r.updateDiscoveryStatus(instance, operatorv1.OperatorStatusTypeProgressing, message,
			operatorv1.ConditionFalse, localv1alpha1.Discovering)
		if err != nil {
			return reconcile.Result{}, err
		}
		return waitForRequeueIfDaemonsNotReady, fmt.Errorf(message)
	}

	message := fmt.Sprintf("successfully running %d out of %d discovery daemons", desiredDaemons, readyDaemons)
	err = r.updateDiscoveryStatus(instance, operatorv1.OperatorStatusTypeAvailable, message,
		operatorv1.ConditionTrue, localv1alpha1.Discovering)
	if err != nil {
		return reconcile.Result{}, err
	}

	r.syncDiskPvConfigMap(instance)

	return reconcile.Result{}, nil
}

func getDiskMakerDiscoveryDSMutateFn(request reconcile.Request,
	tolerations []corev1.Toleration,
	ownerRefs []metav1.OwnerReference,
	nodeSelector *corev1.NodeSelector) func(*appsv1.DaemonSet) error {

	return func(ds *appsv1.DaemonSet) error {
		name := DiskMakerDiscovery

		nodedaemon.MutateAggregatedSpec(
			ds,
			request,
			tolerations,
			ownerRefs,
			nodeSelector,
			name,
		)
		discoveryVolumes, discoveryVolumeMounts := getDiscoveryVolumesAndMounts()
		ds.Spec.Template.Spec.Volumes = discoveryVolumes
		ds.Spec.Template.Spec.Containers[0].VolumeMounts = discoveryVolumeMounts
		ds.Spec.Template.Spec.Containers[0].Env = append(ds.Spec.Template.Spec.Containers[0].Env, getUIDEnvVar())
		ds.Spec.Template.Spec.Containers[0].Image = common.GetDiskMakerImage()
		ds.Spec.Template.Spec.Containers[0].Args = []string{"discover"}

		return nil
	}
}

// updateDiscoveryStatus updates the discovery state with conditions and phase
func (r *ReconcileLocalVolumeDiscovery) updateDiscoveryStatus(instance *localv1alpha1.LocalVolumeDiscovery, conditionType, message string,
	status operatorv1.ConditionStatus, phase localv1alpha1.DiscoveryPhase) error {
	// avoid frequently updating the same status in the CR
	if len(instance.Status.Conditions) < 1 || instance.Status.Conditions[0].Message != message {
		condition := operatorv1.OperatorCondition{
			Type:               conditionType,
			Status:             status,
			Message:            message,
			LastTransitionTime: metav1.Now(),
		}
		newConditions := []operatorv1.OperatorCondition{condition}
		instance.Status.Conditions = newConditions
		instance.Status.Phase = phase
		instance.Status.ObservedGeneration = instance.Generation
		err := r.updateStatus(instance)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *ReconcileLocalVolumeDiscovery) updateStatus(lvd *localv1alpha1.LocalVolumeDiscovery) error {
	err := r.client.Status().Update(context.TODO(), lvd)
	if err != nil {
		return err
	}

	return nil
}

func (r *ReconcileLocalVolumeDiscovery) getDaemonSetStatus(namespace string) (int32, int32, error) {
	existingDS := &appsv1.DaemonSet{}
	err := r.client.Get(context.TODO(), types.NamespacedName{Name: DiskMakerDiscovery, Namespace: namespace}, existingDS)
	if err != nil {
		return 0, 0, err
	}

	return existingDS.Status.DesiredNumberScheduled, existingDS.Status.NumberReady, nil
}

func getDiscoveryVolumesAndMounts() ([]corev1.Volume, []corev1.VolumeMount) {
	hostContainerPropagation := corev1.MountPropagationHostToContainer
	volumeMounts := []corev1.VolumeMount{
		{
			Name:             "local-disks",
			MountPath:        common.GetLocalDiskLocationPath(),
			MountPropagation: &hostContainerPropagation,
		},
		{
			Name:             "device-dir",
			MountPath:        "/dev",
			MountPropagation: &hostContainerPropagation,
		},
		{
			Name:             udevVolName,
			MountPath:        udevPath,
			MountPropagation: &hostContainerPropagation,
		},
	}
	directoryHostPath := corev1.HostPathDirectory
	volumes := []corev1.Volume{
		{
			Name: "local-disks",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: common.GetLocalDiskLocationPath(),
				},
			},
		},
		{
			Name: "device-dir",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/dev",
					Type: &directoryHostPath,
				},
			},
		},
		{
			Name: udevVolName,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{Path: udevPath},
			},
		},
	}

	return volumes, volumeMounts
}

func getOwnerRefs(cr *localv1alpha1.LocalVolumeDiscovery) []metav1.OwnerReference {
	trueVal := true
	return []metav1.OwnerReference{
		{
			APIVersion: localv1alpha1.SchemeGroupVersion.String(),
			Kind:       "LocalVolumeDiscovery",
			Name:       cr.Name,
			UID:        cr.UID,
			Controller: &trueVal,
		},
	}
}

func getUIDEnvVar() corev1.EnvVar {
	return corev1.EnvVar{
		Name: "UID",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.uid",
			},
		},
	}
}

func (r *ReconcileLocalVolumeDiscovery) syncDiskPvConfigMap(cr *localv1alpha1.LocalVolumeDiscovery) {
	pvList := &corev1.PersistentVolumeList{}
	err := r.client.List(context.TODO(), pvList)
	if err != nil {
		klog.Error("failed to get pv list %+v", err)
	}

	// if len(pvList.Items) == 0 {
	// 	return
	// }

	lastDiskPvMap := discovery.NewDiskPVMap()
	for _, pv := range pvList.Items {
		if !strings.Contains(pv.Annotations["pv.kubernetes.io/provisioned-by"], "local-volume-provisioner") {
			continue
		}
		if !lastDiskPvMap.Contains(pv.Labels["kubernetes.io/hostname"], pv.Name) {
			lastDiskPvMap.Add(pv.Labels["kubernetes.io/hostname"], pv.Name, pv.Spec.StorageClassName, pv.Spec.Local.Path)
		}
	}

	cm := &corev1.ConfigMap{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: discovery.DevicePVConfigMapName, Namespace: cr.Namespace}, cm)
	if err != nil {
		if errors.IsNotFound(err) {
			cm = &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:            discovery.DevicePVConfigMapName,
					Namespace:       cr.Namespace,
					OwnerReferences: getOwnerRefs(cr),
				},
			}
			yaml, err := discovery.ToDiskPVMapYAML(lastDiskPvMap)
			if err != nil {
				klog.Infof("Failed to convert data to yaml - %v", err)
				// return nil, err
			}
			cm.Data = map[string]string{
				"devicePvMapConfig": yaml,
			}

			err = r.client.Create(context.TODO(), cm)
			if err != nil {
				klog.Info("Failed to create config map %v", err)
				return
			}

			return
		}
	}

	existingDevicePVMap, err := discovery.ToDiskPVMapObj(cm.Data["devicePvMapConfig"])
	if err != nil {
		klog.Error("failed to parse existing disk pv map data %+v", err)
		return
	}

	klog.Info("Last Device PV Map data - %+v", lastDiskPvMap)
	klog.Info("existing Device PV Map data - %+v", existingDevicePVMap)

	if !discovery.DiskPVMapEqual(*lastDiskPvMap, *existingDevicePVMap) {
		yaml, err := discovery.ToDiskPVMapYAML(lastDiskPvMap)
		if err != nil {
			klog.Infof("Failed to convert data to yaml - %v", err)
			return
		}
		cm.Data = map[string]string{
			"devicePvMapConfig": yaml,
		}

		err = r.client.Update(context.TODO(), cm)
		if err != nil {
			klog.Info("failed to update configmap")
		}
		return
	}

	klog.Info("No new updates in config map")
}
