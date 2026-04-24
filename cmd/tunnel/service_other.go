//go:build !windows

package main

import "fmt"

func installWindowsService(exePath, configPath string) error {
	return fmt.Errorf("windows service installation is only supported on Windows")
}

func uninstallWindowsService() error {
	return fmt.Errorf("windows service uninstallation is only supported on Windows")
}
