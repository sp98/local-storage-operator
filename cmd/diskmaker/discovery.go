package main

import (
	"os"

	"github.com/openshift/local-storage-operator/pkg/diskmaker/discovery"
	"github.com/openshift/local-storage-operator/pkg/localmetrics"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	discoverySerivceName = "localmetrics-lvd"
)

func startDeviceDiscovery(cmd *cobra.Command, args []string) error {
	printVersion()
	// configure local metrics for local volume discovery
	localmetrics.ConfigureCustomMetrics(os.Getenv("WATCH_NAMESPACE"), discoverySerivceName)
	discoveryObj, err := discovery.NewDeviceDiscovery()
	if err != nil {
		return errors.Wrapf(err, "failed to discover devices")
	}
	err = discoveryObj.Start()
	if err != nil {
		return errors.Wrapf(err, "failed to discover devices")
	}
	return nil
}
