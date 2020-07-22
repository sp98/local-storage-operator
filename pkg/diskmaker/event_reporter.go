package diskmaker

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	// LocalVolume events
	ErrorRunningBlockList    = "ErrorRunningBlockList"
	ErrorReadingBlockList    = "ErrorReadingBlockList"
	ErrorListingDeviceID     = "ErrorListingDeviceID"
	ErrorFindingMatchingDisk = "ErrorFindingMatchingDisk"
	ErrorCreatingSymLink     = "ErrorCreatingSymLink"

	FoundMatchingDisk = "FoundMatchingDisk"

	// LocalVolumeDiscovery events
	ErrorCreatingDiscoveryResultObject = "ErrorCreatingDiscoveryResultObject"
	ErrorUpdatingDiscoveryResultObject = "ErrorUpdatingDiscoveryResultObject"
	ErrorListingBlockDevices           = "ErrorListingBlockDevices"

	CreatedDiscoveryResultObject = "CreatedDiscoveryResultObject"
	UpdatedDiscoveredDeviceList  = "UpdatedDiscoveredDeviceList"
)

type DiskEvent struct {
	EventType   string
	EventReason string
	Disk        string
	Message     string
}

func NewEvent(eventReason, message, disk string) *DiskEvent {
	return &DiskEvent{EventReason: eventReason, Disk: disk, Message: message, EventType: corev1.EventTypeWarning}
}

func NewSuccessEvent(eventReason, message, disk string) *DiskEvent {
	return &DiskEvent{EventReason: eventReason, Disk: disk, Message: message, EventType: corev1.EventTypeNormal}
}

type EventReporter struct {
	apiClient      ApiUpdater
	reportedEvents sets.String
}

func NewEventReporter(apiClient ApiUpdater) *EventReporter {
	er := &EventReporter{apiClient: apiClient}
	er.reportedEvents = sets.NewString()
	return er
}

// report function is not thread safe
func (reporter *EventReporter) Report(e *DiskEvent, obj runtime.Object) {
	eventKey := fmt.Sprintf("%s:%s:%s", e.EventReason, e.EventType, e.Disk)
	if reporter.reportedEvents.Has(eventKey) {
		return
	}

	reporter.apiClient.recordEvent(obj, e)
	reporter.reportedEvents.Insert(eventKey)
}
