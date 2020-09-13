package diskmaker

import (
	localv1 "github.com/openshift/local-storage-operator/pkg/apis/local/v1"
	"github.com/openshift/local-storage-operator/pkg/apis/local/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// MockAPIUpdater mocks all the ApiUpdater Commands
type MockAPIUpdater struct {
	events                          []*DiskEvent
	MockGetDiscoveryResult          func(name, namespace string) (*v1alpha1.LocalVolumeDiscoveryResult, error)
	MockCreateDiscoveryResult       func(lvdr *v1alpha1.LocalVolumeDiscoveryResult) error
	MockUpdateDiscoveryResultStatus func(lvdr *v1alpha1.LocalVolumeDiscoveryResult) error
	MockUpdateDiscoveryResult       func(lvdr *v1alpha1.LocalVolumeDiscoveryResult) error
	MockGetLocalVolumeDiscovery     func(name, namespace string) (*v1alpha1.LocalVolumeDiscovery, error)
	MockGetConfigMap                func(name, namespace string) (*v1.ConfigMap, error)
}

var _ ApiUpdater = &MockAPIUpdater{}

func (f *MockAPIUpdater) recordEvent(obj runtime.Object, e *DiskEvent) {
	f.events = append(f.events, e)
}

func (f *MockAPIUpdater) getLocalVolume(lv *localv1.LocalVolume) (*localv1.LocalVolume, error) {
	return lv, nil
}

// GetDiscoveryResult mocks GetDiscoveryResult
func (f *MockAPIUpdater) GetDiscoveryResult(name, namespace string) (*v1alpha1.LocalVolumeDiscoveryResult, error) {
	if f.MockGetDiscoveryResult != nil {
		return f.MockGetDiscoveryResult(name, namespace)
	}

	return &v1alpha1.LocalVolumeDiscoveryResult{}, nil
}

// CreateDiscoveryResult mocks CreateDiscoveryResult
func (f *MockAPIUpdater) CreateDiscoveryResult(lvdr *v1alpha1.LocalVolumeDiscoveryResult) error {
	if f.MockCreateDiscoveryResult != nil {
		return f.MockCreateDiscoveryResult(lvdr)
	}

	return nil
}

// UpdateDiscoveryResultStatus mocks UpdateDiscoveryResultStatus
func (f *MockAPIUpdater) UpdateDiscoveryResultStatus(lvdr *v1alpha1.LocalVolumeDiscoveryResult) error {
	if f.MockUpdateDiscoveryResultStatus != nil {
		return f.MockUpdateDiscoveryResultStatus(lvdr)
	}

	return nil
}

// UpdateDiscoveryResult mocks UpdateDiscoveryResult
func (f *MockAPIUpdater) UpdateDiscoveryResult(lvdr *v1alpha1.LocalVolumeDiscoveryResult) error {
	if f.MockUpdateDiscoveryResult != nil {
		return f.MockUpdateDiscoveryResult(lvdr)
	}

	return nil
}

// GetLocalVolumeDiscovery mocks GetLocalVolumeDiscovery
func (f *MockAPIUpdater) GetLocalVolumeDiscovery(name, namespace string) (*v1alpha1.LocalVolumeDiscovery, error) {
	if f.MockGetLocalVolumeDiscovery != nil {
		return f.MockGetLocalVolumeDiscovery(name, namespace)
	}

	return &v1alpha1.LocalVolumeDiscovery{}, nil
}

// GetConfigMap mock MockGetConfigMap
func (f *MockAPIUpdater) GetConfigMap(name, namespace string) (*v1.ConfigMap, error) {
	if f.MockGetConfigMap != nil {
		return f.MockGetConfigMap(name, namespace)
	}

	return &v1.ConfigMap{}, nil
}
