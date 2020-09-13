package discovery

import (
	"fmt"
	"reflect"

	"gopkg.in/yaml.v2"
)

const (
	// DevicePVConfigMapName is the config map name
	DevicePVConfigMapName = "device-pv-map"
)

// DiskPVMapValues is
type DiskPVMapValues struct {
	PVName           string
	StorageClassName string
	DevicePath       string
}

// DiskPVMap is
type DiskPVMap struct {
	Data map[string][]DiskPVMapValues
}

// NewDiskPVMap returns
func NewDiskPVMap() *DiskPVMap {
	return &DiskPVMap{
		Data: map[string][]DiskPVMapValues{},
	}
}

// Contains checks
func (m *DiskPVMap) Contains(nodeName, pvName string) bool {
	if values, ok := m.Data[nodeName]; ok {
		for _, val := range values {
			if val.PVName == pvName {
				return true
			}
		}
	}
	return false
}

// GetDevicePVMap returns
func (m *DiskPVMap) GetDevicePVMap(nodeName, devicePath string) *DiskPVMapValues {
	if values, ok := m.Data[nodeName]; ok {
		for _, val := range values {
			if val.DevicePath == devicePath {
				return &val
			}
		}
	}
	return nil
}

// Add adds
func (m *DiskPVMap) Add(nodeName, pvName, storageClassName, devicePath string) {
	newValues := DiskPVMapValues{
		PVName:           pvName,
		StorageClassName: storageClassName,
		DevicePath:       devicePath,
	}
	if _, ok := m.Data[nodeName]; !ok {
		m.Data[nodeName] = make([]DiskPVMapValues, 0)
	}

	m.Data[nodeName] = append(m.Data[nodeName], newValues)
}

// ToDiskPVMapYAML returns yaml representation of disk pv map
func ToDiskPVMapYAML(m *DiskPVMap) (string, error) {
	y, err := yaml.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("error marshaling to yaml: %v", err)
	}
	return string(y), nil
}

// ToDiskPVMapObj returns
func ToDiskPVMapObj(devicePVMapString string) (*DiskPVMap, error) {
	var devicePvMap DiskPVMap
	err := yaml.Unmarshal([]byte(devicePVMapString), &devicePvMap)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal devicePVMap:  %+v", err)
	}

	return &devicePvMap, nil
}

// DiskPVMapEqual checks if two diskPVMaps are equal
func DiskPVMapEqual(diskPVMap1, diskPVMap2 DiskPVMap) bool {
	if reflect.DeepEqual(diskPVMap1, diskPVMap2) {
		return true
	}
	return false
}
