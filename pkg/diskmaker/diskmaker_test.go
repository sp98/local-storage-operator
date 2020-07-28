package diskmaker

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	localv1 "github.com/openshift/local-storage-operator/pkg/apis/local/v1"
)

func TestFindMatchingDisk(t *testing.T) {
	d := getFakeDiskMaker("/tmp/foo", "/mnt/local-storage")
	deviceSet := d.findNewDisks(getRawOutput())
	if len(deviceSet) != 5 {
		t.Errorf("expected 7 devices got %d", len(deviceSet))
	}
	diskConfig := &DiskConfig{
		Disks: map[string]*Disks{
			"foo": &Disks{
				DevicePaths: []string{"xyz"},
			},
		},
	}
	allDiskIds := getDeiveIDs()
	deviceMap, err := d.findMatchingDisks(diskConfig, deviceSet, allDiskIds)
	if err != nil {
		t.Fatalf("error finding matchin device %v", err)
	}
	if len(deviceMap) != 0 {
		t.Errorf("expected 0 elements in map got %d", len(deviceMap))
	}
}

func TestLoadConfig(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "diskmaker")
	if err != nil {
		t.Fatalf("error creating temp directory : %v", err)
	}

	defer os.RemoveAll(tempDir)
	diskConfig := &DiskConfig{
		Disks: map[string]*Disks{
			"foo": &Disks{
				DevicePaths: []string{"xyz"},
			},
		},
		OwnerName:       "foobar",
		OwnerNamespace:  "default",
		OwnerKind:       localv1.LocalVolumeKind,
		OwnerUID:        "foobar",
		OwnerAPIVersion: localv1.SchemeGroupVersion.String(),
	}
	yaml, err := diskConfig.ToYAML()
	if err != nil {
		t.Fatalf("error marshalling yaml : %v", err)
	}
	filename := filepath.Join(tempDir, "config")
	err = ioutil.WriteFile(filename, []byte(yaml), 0755)
	if err != nil {
		t.Fatalf("error writing yaml to disk : %v", err)
	}

	d := getFakeDiskMaker(filename, "/mnt/local-storage")
	diskConfigFromDisk, err := d.loadConfig()
	if err != nil {
		t.Fatalf("error loading diskconfig from disk : %v", err)
	}
	if diskConfigFromDisk == nil {
		t.Fatalf("expected a diskconfig got nil")
	}
	if d.localVolume == nil {
		t.Fatalf("expected localvolume got nil")
	}

	if d.localVolume.Name != diskConfig.OwnerName {
		t.Fatalf("expected owner name to be %s got %s", diskConfig.OwnerName, d.localVolume.Name)
	}
}

func getFakeDiskMaker(configLocation, symlinkLocation string) *DiskMaker {
	d := &DiskMaker{configLocation: configLocation, symlinkLocation: symlinkLocation}
	d.apiClient = &MockAPIUpdater{}
	d.eventSync = NewEventReporter(d.apiClient)
	return d
}

func getDeiveIDs() []string {
	return []string{
		"/dev/disk/by-id/xyz",
	}
}
