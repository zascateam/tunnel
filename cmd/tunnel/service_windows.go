//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"
)

var (
	modadvapi32       = syscall.NewLazyDLL("advapi32.dll")
	procCreateService = modadvapi32.NewProc("CreateServiceW")
	procDeleteService = modadvapi32.NewProc("DeleteService")
	procOpenSCManager = modadvapi32.NewProc("OpenSCManagerW")
	procOpenService   = modadvapi32.NewProc("OpenServiceW")
	procCloseService  = modadvapi32.NewProc("CloseServiceHandle")
	procStartService  = modadvapi32.NewProc("StartServiceW")
)

const (
	SC_MANAGER_CREATE_SERVICE = 0x0002
	SERVICE_WIN32_OWN_PROCESS = 0x00000010
	SERVICE_AUTO_START        = 0x00000002
	SERVICE_ERROR_NORMAL      = 0x00000001
	SERVICE_ALL_ACCESS        = 0x000F01FF
	DELETE                    = 0x00010000
)

func installWindowsService(exePath, configPath string) error {
	mgr, _, err := procOpenSCManager.Call(
		0,
		0,
		uintptr(SC_MANAGER_CREATE_SERVICE),
	)
	if mgr == 0 {
		return fmt.Errorf("failed to open SC manager: %v", err)
	}
	defer procCloseService.Call(mgr)

	name, _ := syscall.UTF16PtrFromString(serviceName)
	display, _ := syscall.UTF16PtrFromString(serviceDisplay)
	desc, _ := syscall.UTF16PtrFromString(serviceDesc)
	binaryPath, _ := syscall.UTF16PtrFromString(fmt.Sprintf(`"%s" run -config "%s"`, exePath, configPath))

	svc, _, err := procCreateService.Call(
		mgr,
		uintptr(unsafe.Pointer(name)),
		uintptr(unsafe.Pointer(display)),
		uintptr(SERVICE_ALL_ACCESS),
		uintptr(SERVICE_WIN32_OWN_PROCESS),
		uintptr(SERVICE_AUTO_START),
		uintptr(SERVICE_ERROR_NORMAL),
		uintptr(unsafe.Pointer(binaryPath)),
		0, 0, 0, 0, 0,
	)
	if svc == 0 {
		return fmt.Errorf("failed to create service: %v", err)
	}
	defer procCloseService.Call(svc)

	descCmd := exec.Command("sc", "description", serviceName, desc)
	descCmd.Run()

	procStartService.Call(svc, 0, 0)

	return nil
}

func uninstallWindowsService() error {
	mgr, _, err := procOpenSCManager.Call(0, 0, uintptr(SC_MANAGER_CREATE_SERVICE))
	if mgr == 0 {
		return fmt.Errorf("failed to open SC manager: %v", err)
	}
	defer procCloseService.Call(mgr)

	name, _ := syscall.UTF16PtrFromString(serviceName)
	svc, _, err := procOpenService.Call(mgr, uintptr(unsafe.Pointer(name)), uintptr(DELETE))
	if svc == 0 {
		return fmt.Errorf("failed to open service: %v", err)
	}
	defer procCloseService.Call(svc)

	stopCmd := exec.Command("net", "stop", serviceName)
	stopCmd.Run()

	ret, _, _ := procDeleteService.Call(svc)
	if ret == 0 {
		return fmt.Errorf("failed to delete service")
	}

	return nil
}
