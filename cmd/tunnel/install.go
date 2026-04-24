package main

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	serviceName     = "ZASCA Edge Service"
	serviceDisplay  = "ZASCA Edge Service"
	serviceDesc     = "Provides secure tunnel connectivity for ZASCA cloud desktop management platform"
	installDir      = `C:\Program Files\ZASCA`
	configDir       = `C:\ProgramData\ZASCA`
	configFile      = `tunnel.yaml`
	serviceBinary   = `zasca-tunnel.exe`
)

func runInstall(token, server string) error {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	cfgPath := filepath.Join(configDir, configFile)
	cfgContent := fmt.Sprintf("token: %s\nserver: %s\n", token, server)
	if err := os.WriteFile(cfgPath, []byte(cfgContent), 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	if err := installWindowsService(exePath, cfgPath); err != nil {
		return fmt.Errorf("failed to install service: %w", err)
	}

	fmt.Println("ZASCA Edge Service installed successfully")
	fmt.Printf("Config written to: %s\n", cfgPath)
	fmt.Println("Service will auto-start on boot")
	return nil
}

func runUninstall() error {
	if err := uninstallWindowsService(); err != nil {
		return fmt.Errorf("failed to uninstall service: %w", err)
	}

	cfgPath := filepath.Join(configDir, configFile)
	os.Remove(cfgPath)

	fmt.Println("ZASCA Edge Service uninstalled successfully")
	return nil
}
