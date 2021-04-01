package localmetrics

import (
	"context"

	cmetrics "github.com/openshift/operator-custom-metrics/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	log          = logf.Log.WithName("localmetrics")
	cmetricsHost = "0.0.0.0"
	cmetricsPort = "8383"
)

var (
	metricPVProvisionedByLocalVolume = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "local_volume_provisioned_pvs",
		Help: "Report how many persistent volumes have been provisoned by Local Volume Operator",
	}, []string{"nodeName", "storageClass"})

	metricPVProvisionedByLocalVolumeSet = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "local_volume_set_provisioned_pvs",
		Help: "Report how many persistent volumes have been provisoned by Local Volume Operator",
	}, []string{"nodeName", "storageClass"})

	metricDiscoveredDevicesByLocalVolumeDiscovery = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "local_volume_discovery_discovered_disks",
		Help: "Report how many disks were discoverd via the Local Volume Discovery Operator",
	}, []string{"nodeName"})

	metricUnmatchedDevicesByLocalVolumeSet = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "local_volume_unmatched_disks",
		Help: "Report how many disks didn't match the Local Volume Set filter",
	}, []string{"nodeName", "storageClass"})

	metricOrphanSynlinksByLocalVolumeSet = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "local_volume_set_orphaned_symlinks",
		Help: "Report how many synlinks become orphan after updating the Local Volume Set filter",
	}, []string{"nodeName", "storageClass"})

	MetricsList = []prometheus.Collector{
		metricPVProvisionedByLocalVolume,
		metricPVProvisionedByLocalVolumeSet,
		metricDiscoveredDevicesByLocalVolumeDiscovery,
		metricUnmatchedDevicesByLocalVolumeSet,
		metricOrphanSynlinksByLocalVolumeSet,
	}
)

func SetDiscoveredDevicesMetrics(nodeName string, deviceCount int) {
	metricDiscoveredDevicesByLocalVolumeDiscovery.
		With(prometheus.Labels{"nodeName": nodeName}).
		Set(float64((deviceCount)))
}

func SetLVSProvisionedPVs(nodeName, sc string, pvCount int) {
	metricPVProvisionedByLocalVolumeSet.
		With(prometheus.Labels{"nodeName": nodeName, "storageClass": sc}).
		Set(float64((pvCount)))
}

func SetLVProvisionedPVs(nodeName, sc string, pvCount int) {
	metricPVProvisionedByLocalVolume.
		With(prometheus.Labels{"nodeName": nodeName, "storageClass": sc}).
		Set(float64((pvCount)))
}

func SetLVSUnmatchedDevices(nodeName, sc string, deviceCount int) {
	metricUnmatchedDevicesByLocalVolumeSet.
		With(prometheus.Labels{"nodeName": nodeName, "storageClass": sc}).
		Set(float64((deviceCount)))
}

func SetLVSOrphanSymlinks(nodeName, sc string, deviceCount int) {
	metricOrphanSynlinksByLocalVolumeSet.
		With(prometheus.Labels{"nodeName": nodeName, "storageClass": sc}).
		Set(float64((deviceCount)))
}

func ConfigureCustomMetrics(namespace, servicename string) {
	metricsServer := cmetrics.NewBuilder(namespace, servicename).
		WithPort(cmetricsPort).
		WithPath(cmetricsHost).
		WithCollectors(MetricsList).
		WithServiceMonitor().
		GetConfig()

	if err := cmetrics.ConfigureMetrics(context.TODO(), *metricsServer); err != nil {
		log.Error(err, "Failed to configure custom metrics")
	}
}
