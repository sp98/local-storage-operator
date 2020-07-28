package discovery

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/openshift/local-storage-operator/pkg/apis/local/v1alpha1"
	"github.com/openshift/local-storage-operator/pkg/internal"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	testData1 = `NAME="sda" ROTA="1" TYPE="disk" SIZE="62914560000" MODEL="VBOX HARDDISK" VENDOR="ATA" RO="1" RM="0" STATE="running" FSTYPE="" SERIAL=""
NAME="sda1" ROTA="1" TYPE="part" SIZE="62913494528" MODEL="" VENDOR="" RO="1" RM="0" STATE="" FSTYPE="" SERIAL=""
`
	testData2 = `NAME="sdc" ROTA="1" TYPE="disk" SIZE="62914560000" MODEL="VBOX HARDDISK" VENDOR="ATA" RO="0" RM="1" STATE="running" FSTYPE="ext4" SERIAL=""
NAME="sdc3" ROTA="1" TYPE="part" SIZE="62913494528" MODEL="" VENDOR="" RO="0" RM="1" STATE="" FSTYPE="ext4" SERIAL=""
`
)

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	fmt.Fprintf(os.Stdout, os.Getenv("STDOUT"))
	os.Exit(0)
}

func TestIgnoreDevices(t *testing.T) {
	testcases := []struct {
		label        string
		blockDevice  internal.BlockDevice
		fakeGlobfunc func(string) ([]string, error)
		expected     bool
		errMessage   error
	}{
		{
			label: "Case 1",
			blockDevice: internal.BlockDevice{
				Name:     "sdb",
				KName:    "sdb",
				ReadOnly: "0",
				State:    "running",
				Type:     "disk",
			},
			fakeGlobfunc: func(name string) ([]string, error) {
				return []string{"removable", "subsytem"}, nil
			},
			expected:   false,
			errMessage: fmt.Errorf("ignored wrong device"),
		},
		{
			label: "Case 2",
			blockDevice: internal.BlockDevice{
				Name:     "sdb",
				KName:    "sdb",
				ReadOnly: "1",
				State:    "running",
				Type:     "disk",
			},
			fakeGlobfunc: func(name string) ([]string, error) {
				return []string{"removable", "subsytem"}, nil
			},
			expected:   true,
			errMessage: fmt.Errorf("failed to ignore read only device"),
		},
		{
			label: "Case 3",
			blockDevice: internal.BlockDevice{
				Name:     "sdb",
				KName:    "sdb",
				ReadOnly: "0",
				State:    "suspended",
				Type:     "disk",
			},
			fakeGlobfunc: func(name string) ([]string, error) {
				return []string{"removable", "subsytem"}, nil
			},
			expected:   true,
			errMessage: fmt.Errorf("ignored wrong suspended device"),
		},
		{
			label: "Case 4",
			blockDevice: internal.BlockDevice{
				Name:     "sdb",
				KName:    "sdb",
				ReadOnly: "0",
				State:    "running",
				Type:     "disk",
			},
			fakeGlobfunc: func(name string) ([]string, error) {
				return []string{"removable", "subsytem", "sdb"}, nil
			},
			expected:   true,
			errMessage: fmt.Errorf("failed to ignore root device with children"),
		},
	}

	for _, tc := range testcases {
		internal.FilePathGlob = tc.fakeGlobfunc
		defer func() {
			internal.FilePathGlob = filepath.Glob
		}()

		actual := ignoreDevices(tc.blockDevice)
		assert.Equalf(t, tc.expected, actual, "[%s]: %s", tc.label, tc.errMessage)
	}
}

func TestValidBlockDevices(t *testing.T) {
	testcases := []struct {
		label                        string
		blockDevices                 []internal.BlockDevice
		fakeExecCmdOutput            string
		fakeGlobfunc                 func(string) ([]string, error)
		expectedDiscoveredDeviceSize int
		errMessage                   error
	}{
		{
			label: "Case 1",
			fakeExecCmdOutput: `NAME="sda" KNAME="sda" ROTA="1" TYPE="disk" SIZE="62914560000" MODEL="VBOX HARDDISK" VENDOR="ATA" RO="1" RM="0" STATE="running" FSTYPE="" SERIAL=""` + "\n" +
				`NAME="sda1" KNAME="sda1" ROTA="1" TYPE="part" SIZE="62913494528" MODEL="" VENDOR="" RO="0" RM="0" STATE="" FSTYPE="" SERIAL=""`,
			fakeGlobfunc: func(name string) ([]string, error) {
				return []string{"removable", "subsytem"}, nil
			},
			expectedDiscoveredDeviceSize: 1,
			errMessage:                   fmt.Errorf("failed to ignore readonly device sda"),
		},
		{
			label: "Case 2",
			fakeExecCmdOutput: `NAME="sda" KNAME="sda" ROTA="1" TYPE="disk" SIZE="62914560000" MODEL="VBOX HARDDISK" VENDOR="ATA" RO="0" RM="0" STATE="running" FSTYPE="" SERIAL=""` + "\n" +
				`NAME="sda1" KNAME="sda1" ROTA="1" TYPE="part" SIZE="62913494528" MODEL="" VENDOR="" RO="0" RM="0" STATE="" FSTYPE="" SERIAL=""`,
			fakeGlobfunc: func(name string) ([]string, error) {
				return []string{"removable", "subsytem", "sda"}, nil
			},
			expectedDiscoveredDeviceSize: 1,
			errMessage:                   fmt.Errorf("failed to ignore root device sda with partition"),
		},
		{
			label: "Case 3",
			fakeExecCmdOutput: `NAME="sda" KNAME="sda" ROTA="1" TYPE="loop" SIZE="62914560000" MODEL="VBOX HARDDISK" VENDOR="ATA" RO="0" RM="0" STATE="running" FSTYPE="" SERIAL=""` + "\n" +
				`NAME="sda1" KNAME="sda1" ROTA="1" TYPE="part" SIZE="62913494528" MODEL="" VENDOR="" RO="0" RM="0" STATE="" FSTYPE="" SERIAL=""`,
			fakeGlobfunc: func(name string) ([]string, error) {
				return []string{"removable", "subsytem"}, nil
			},
			expectedDiscoveredDeviceSize: 1,
			errMessage:                   fmt.Errorf("failed to ignore device sda with type loop"),
		},
		{
			label: "Case 4",
			fakeExecCmdOutput: `NAME="sda" KNAME="sda" ROTA="1" TYPE="disk" SIZE="62914560000" MODEL="VBOX HARDDISK" VENDOR="ATA" RO="0" RM="0" STATE="running" FSTYPE="" SERIAL=""` + "\n" +
				`NAME="sda1" KNAME="sda1" ROTA="1" TYPE="part" SIZE="62913494528" MODEL="" VENDOR="" RO="0" RM="0" STATE="suspended" FSTYPE="" SERIAL=""`,
			fakeGlobfunc: func(name string) ([]string, error) {
				return []string{"removable", "subsytem"}, nil
			},
			expectedDiscoveredDeviceSize: 1,
			errMessage:                   fmt.Errorf("failed to ignore child device sda1 in suspended state"),
		},
	}

	for _, tc := range testcases {
		internal.ExecResult = tc.fakeExecCmdOutput
		internal.ExecCommand = internal.FakeExecCommand
		internal.FilePathGlob = tc.fakeGlobfunc
		defer func() {
			internal.FilePathGlob = filepath.Glob
			internal.ExecCommand = exec.Command
		}()
		actual, err := getValidBlockDevices()
		assert.NoError(t, err)
		assert.Equalf(t, tc.expectedDiscoveredDeviceSize, len(actual), "[%s]: %s", tc.label, tc.errMessage)
	}
}

func TestGetDiscoveredDevices(t *testing.T) {
	testcases := []struct {
		label               string
		blockDevices        []internal.BlockDevice
		expected            []v1alpha1.DiscoveredDevice
		fakeGlobfunc        func(string) ([]string, error)
		fakeEvalSymlinkfunc func(string) (string, error)
	}{
		{
			label: "Case 1",
			blockDevices: []internal.BlockDevice{
				{
					Name:       "sdb",
					KName:      "sdb",
					FSType:     "ext4",
					Type:       "disk",
					Size:       "62914560000",
					Model:      "VBOX HARDDISK",
					Vendor:     "ATA",
					Serial:     "DEVICE_SERIAL_NUMBER",
					Rotational: "1",
					ReadOnly:   "0",
					Removable:  "0",
					State:      "running",
				},
			},
			expected: []v1alpha1.DiscoveredDevice{
				{
					DeviceID: "/dev/disk/by-id/sdb",
					Path:     "/dev/sdb",
					Model:    "VBOX HARDDISK",
					Type:     "disk",
					Vendor:   "ATA",
					Serial:   "DEVICE_SERIAL_NUMBER",
					Size:     resource.MustParse("62914560000"),
					Property: "Rotational",
					FSType:   "ext4",
					Status:   v1alpha1.DeviceStatus{State: "NotAvailable"},
				},
			},
			fakeGlobfunc: func(name string) ([]string, error) {
				return []string{"/dev/disk/by-id/sdb"}, nil
			},
			fakeEvalSymlinkfunc: func(path string) (string, error) {
				return "/dev/disk/by-id/sdb", nil
			},
		},

		{
			label: "Case 2",
			blockDevices: []internal.BlockDevice{
				{
					Name:       "sda1",
					KName:      "sda1",
					FSType:     "ext4",
					Type:       "part",
					Size:       "62913494528",
					Model:      "",
					Vendor:     "",
					Serial:     "",
					Rotational: "0",
					ReadOnly:   "0",
					Removable:  "0",
					State:      "running",
				},
			},
			expected: []v1alpha1.DiscoveredDevice{
				{
					DeviceID: "/dev/disk/by-id/sda1",
					Path:     "/dev/sda1",
					Model:    "",
					Type:     "part",
					Vendor:   "",
					Serial:   "",
					Size:     resource.MustParse("62913494528"),
					Property: "NonRotational",
					FSType:   "ext4",
					Status:   v1alpha1.DeviceStatus{State: "NotAvailable"},
				},
			},
			fakeGlobfunc: func(name string) ([]string, error) {
				return []string{"/dev/disk/by-id/sda1"}, nil
			},
			fakeEvalSymlinkfunc: func(path string) (string, error) {
				return "/dev/disk/by-id/sda1", nil
			},
		},
	}

	for _, tc := range testcases {
		internal.FilePathGlob = tc.fakeGlobfunc
		internal.FilePathEvalSymLinks = tc.fakeEvalSymlinkfunc
		defer func() {
			internal.FilePathGlob = filepath.Glob
			internal.FilePathEvalSymLinks = filepath.EvalSymlinks
		}()

		actual := getDiscoverdDevices(tc.blockDevices)
		for i := 0; i < len(tc.expected); i++ {
			assert.Equalf(t, tc.expected[i].DeviceID, actual[i].DeviceID, "[%s: Discovered Device: %d]: invalid device ID", tc.label, i+1)
			assert.Equalf(t, tc.expected[i].Path, actual[i].Path, "[%s: Discovered Device: %d]: invalid device path", tc.label, i+1)
			assert.Equalf(t, tc.expected[i].Model, actual[i].Model, "[%s: Discovered Device: %d]: invalid device model", tc.label, i+1)
			assert.Equalf(t, tc.expected[i].Type, actual[i].Type, "[%s: Discovered Device: %d]: invalid device type", tc.label, i+1)
			assert.Equalf(t, tc.expected[i].Vendor, actual[i].Vendor, "[%s: Discovered Device: %d]: invalid device vendor", tc.label, i+1)
			assert.Equalf(t, tc.expected[i].Serial, actual[i].Serial, "[%s: Discovered Device: %d]: invalid device serial", tc.label, i+1)
			assert.Equalf(t, tc.expected[i].Size, actual[i].Size, "[%s: Discovered Device: %d]: invalid device size", tc.label, i+1)
			assert.Equalf(t, tc.expected[i].Property, actual[i].Property, "[%s: Discovered Device: %d]: invalid device property", tc.label, i+1)
			assert.Equalf(t, tc.expected[i].FSType, actual[i].FSType, "[%s: Discovered Device: %d]: invalid device filesystem", tc.label, i+1)
			assert.Equalf(t, tc.expected[i].Status, actual[i].Status, "[%s: Discovered Device: %d]: invalid device status", tc.label, i+1)
		}
	}
}

func TestParseDeviceType(t *testing.T) {
	testcases := []struct {
		label    string
		input    string
		expected v1alpha1.DeviceType
	}{
		{
			label:    "Case 1",
			input:    "disk",
			expected: v1alpha1.RawDisk,
		},
		{
			label:    "Case 1",
			input:    "part",
			expected: v1alpha1.Partition,
		},
		{
			label:    "Case 3",
			input:    "loop",
			expected: "",
		},
	}

	for _, tc := range testcases {
		actual := parseDeviceType(tc.input)
		assert.Equalf(t, tc.expected, actual, "[%s]: failed to parse device type", tc.label)
	}
}

func TestParseDeviceProperty(t *testing.T) {
	testcases := []struct {
		label    string
		input    string
		expected v1alpha1.DeviceMechanicalProperty
	}{
		{
			label:    "Case 1",
			input:    "1",
			expected: v1alpha1.Rotational,
		},
		{
			label:    "Case 1",
			input:    "0",
			expected: v1alpha1.NonRotational,
		},
		{
			label:    "Case 3",
			input:    "2",
			expected: "",
		},
	}

	for _, tc := range testcases {
		actual := parseDeviceProperty(tc.input)
		assert.Equalf(t, tc.expected, actual, "[%s]: failed to parse device mechanical property", tc.label)
	}
}
