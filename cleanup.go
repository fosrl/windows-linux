package main

import (
	"github.com/fosrl/newt/logger"
	"github.com/fosrl/olm/blockpage"
)

// uninstallCleanup removes what tunnels added to the system outside the
// install directory, so it does not outlive the package: the DNS block page
// CA in the system trust store and browser policies, and its key. Run by the
// package's removal scripts (Linux) and the MSI (Windows) with root or SYSTEM
// rights, after the service has stopped.
func uninstallCleanup() error {
	if err := blockpage.Uninstall(blockpage.DefaultDir()); err != nil {
		logger.Error("Uninstall cleanup: %v", err)
		return err
	}
	logger.Info("Uninstall cleanup done")
	return nil
}
