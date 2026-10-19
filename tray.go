//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	cmdStatus    = 1001
	cmdAutostart = 1002
	cmdLog       = 1003
	cmdExit      = 1004
)

var nid notifyIconData

func addTrayIcon(hwnd syscall.Handle) {
	icon, _, _ := pLoadIcon.Call(0, uintptr(idiApplication))
	nid = notifyIconData{
		CbSize:           uint32(unsafe.Sizeof(notifyIconData{})),
		HWnd:             hwnd,
		UID:              1,
		UFlags:           nifMessage | nifIcon | nifTip,
		UCallbackMessage: wmTrayIcon,
		HIcon:            syscall.Handle(icon),
	}
	setTip("Pageant-Bruecke zum Bitwarden SSH-Agent")
	pShellNotifyIcon.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))
}

func setTip(s string) {
	src := syscall.StringToUTF16(s)
	for i := range nid.SzTip {
		nid.SzTip[i] = 0
	}
	n := len(src)
	if n > len(nid.SzTip) {
		n = len(nid.SzTip)
	}
	copy(nid.SzTip[:], src[:n])
	nid.SzTip[len(nid.SzTip)-1] = 0
}

func updateTip() {
	state := "Agent erreichbar"
	if !agentReachable() {
		state = "Agent NICHT erreichbar"
	}
	setTip(fmt.Sprintf("Pageant-Bruecke: %s, %d Anfragen", state, reqCount.Load()))
	pShellNotifyIcon.Call(nimModify, uintptr(unsafe.Pointer(&nid)))
}

func removeTrayIcon() {
	pShellNotifyIcon.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
}

func showMenu(hwnd syscall.Handle) {
	menu, _, _ := pCreatePopupMenu.Call()
	if menu == 0 {
		return
	}
	defer pDestroyMenu.Call(menu)

	status := "Bitwarden-Agent: erreichbar"
	if !agentReachable() {
		status = "Bitwarden-Agent: nicht erreichbar"
	}
	appendItem(menu, mfString|1 /*MF_GRAYED*/, cmdStatus, status)
	appendItem(menu, mfString|1, 0, fmt.Sprintf("Bearbeitete Anfragen: %d", reqCount.Load()))
	pAppendMenu.Call(menu, mfSeparator, 0, 0)

	flags := uintptr(mfString | mfUnchecked)
	if autostartEnabled() {
		flags = mfString | mfChecked
	}
	appendItem(menu, flags, cmdAutostart, "Mit Windows starten")
	appendItem(menu, mfString, cmdLog, "Protokoll anzeigen")
	pAppendMenu.Call(menu, mfSeparator, 0, 0)
	appendItem(menu, mfString, cmdExit, "Beenden")

	var pt point
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	pSetForegroundWin.Call(uintptr(hwnd))
	pTrackPopupMenu.Call(menu, tpmRightButton|tpmBottomAlign,
		uintptr(pt.X), uintptr(pt.Y), 0, uintptr(hwnd), 0)
	pPostMessage.Call(uintptr(hwnd), wmNull, 0, 0)
}

func appendItem(menu uintptr, flags uintptr, id uintptr, text string) {
	t := utf16Ptr(text)
	pAppendMenu.Call(menu, flags, id, uintptr(unsafe.Pointer(t)))
}

func handleCommand(hwnd syscall.Handle, id uintptr) {
	switch id {
	case cmdAutostart:
		want := !autostartEnabled()
		if err := setAutostart(want); err != nil {
			messageBox("Pageant-Bruecke",
				"Autostart konnte nicht geaendert werden:\n"+fmtErr(err), mbOK|mbIconWarn)
			return
		}
		if want {
			logf("Autostart aktiviert")
		} else {
			logf("Autostart deaktiviert")
		}
	case cmdLog:
		path, err := writeLogFile()
		if err != nil {
			messageBox("Pageant-Bruecke", "Protokoll konnte nicht geschrieben werden:\n"+fmtErr(err), mbOK|mbIconWarn)
			return
		}
		verb := utf16Ptr("open")
		file := utf16Ptr("notepad.exe")
		args := utf16Ptr(`"` + path + `"`)
		pShellExecute.Call(0,
			uintptr(unsafe.Pointer(verb)),
			uintptr(unsafe.Pointer(file)),
			uintptr(unsafe.Pointer(args)), 0, swShow)
	case cmdExit:
		pDestroyWindow.Call(uintptr(hwnd))
	}
}
