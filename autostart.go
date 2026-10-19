//go:build windows

package main

import (
	"os"
	"syscall"
	"unsafe"
)

const (
	runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValue   = "BwPageantBridge"
)

func openRunKey(access uint32) (syscall.Handle, error) {
	var h syscall.Handle
	var disp uint32
	path := utf16Ptr(runKeyPath)
	ret, _, _ := pRegCreateKeyEx.Call(
		uintptr(hkeyCurrentUser),
		uintptr(unsafe.Pointer(path)),
		0, 0, regOptionNonVolatile,
		uintptr(access), 0,
		uintptr(unsafe.Pointer(&h)),
		uintptr(unsafe.Pointer(&disp)))
	if ret != errorSuccess {
		return 0, syscall.Errno(ret)
	}
	return h, nil
}

func autostartEnabled() bool {
	h, err := openRunKey(keyRead)
	if err != nil {
		return false
	}
	defer pRegCloseKey.Call(uintptr(h))

	var typ uint32
	var size uint32
	name := utf16Ptr(runValue)
	ret, _, _ := pRegQueryValueEx.Call(
		uintptr(h), uintptr(unsafe.Pointer(name)), 0,
		uintptr(unsafe.Pointer(&typ)), 0,
		uintptr(unsafe.Pointer(&size)))
	return ret == errorSuccess
}

func setAutostart(on bool) error {
	h, err := openRunKey(keyWrite)
	if err != nil {
		return err
	}
	defer pRegCloseKey.Call(uintptr(h))

	name := utf16Ptr(runValue)
	if !on {
		ret, _, _ := pRegDeleteValue.Call(uintptr(h), uintptr(unsafe.Pointer(name)))
		if ret != errorSuccess {
			return syscall.Errno(ret)
		}
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	// Anfuehrungszeichen, damit Leerzeichen im Pfad nicht stoeren.
	data, err := syscall.UTF16FromString(`"` + exe + `"`)
	if err != nil {
		return err
	}
	ret, _, _ := pRegSetValueEx.Call(
		uintptr(h), uintptr(unsafe.Pointer(name)), 0, regSZ,
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)*2))
	if ret != errorSuccess {
		return syscall.Errno(ret)
	}
	return nil
}
