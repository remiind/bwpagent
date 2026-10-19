//go:build windows

package main

import (
	"runtime"
	"syscall"
	"time"
	"unsafe"
)

const (
	pageantClass = "Pageant"
	pageantTitle = "Pageant"
)

var (
	wndProcCallback = syscall.NewCallback(wndProc)
	mainWindow      syscall.Handle
)

func wndProc(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case wmCopyData:
		return handleCopyData(lParam)

	case wmTrayIcon:
		switch lParam {
		case wmRButtonUp, wmLButtonDblClk:
			showMenu(hwnd)
		}
		return 0

	case wmCommand:
		handleCommand(hwnd, wParam&0xffff)
		return 0

	case wmClose:
		pDestroyWindow.Call(uintptr(hwnd))
		return 0

	case wmDestroy:
		removeTrayIcon()
		pPostQuitMessage.Call(0)
		return 0
	}
	ret, _, _ := pDefWindowProc.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return ret
}

func main() {
	// Die Fensterprozedur und die Nachrichtenschleife muessen auf demselben
	// Betriebssystem-Thread laufen.
	runtime.LockOSThread()

	// Nur eine Instanz: Existiert bereits ein Pageant-Fenster, laeuft entweder
	// das echte Pageant oder eine zweite Bruecke. Beides waere ein Konflikt.
	if h, _, _ := pFindWindow.Call(
		uintptr(unsafe.Pointer(utf16Ptr(pageantClass))),
		uintptr(unsafe.Pointer(utf16Ptr(pageantTitle)))); h != 0 {
		messageBox("Pageant-Bruecke",
			"Es laeuft bereits ein Pageant-kompatibler Agent (Pageant selbst oder eine zweite Instanz dieser Bruecke).\n\n"+
				"Bitte diesen zuerst beenden.", mbOK|mbIconWarn)
		return
	}

	inst, _, _ := pGetModuleHandle.Call(0)
	cursor, _, _ := pLoadCursor.Call(0, uintptr(idcArrow))
	icon, _, _ := pLoadIcon.Call(0, uintptr(idiApplication))

	className := utf16Ptr(pageantClass)
	wc := wndClassEx{
		CbSize:        uint32(unsafe.Sizeof(wndClassEx{})),
		LpfnWndProc:   wndProcCallback,
		HInstance:     syscall.Handle(inst),
		HIcon:         syscall.Handle(icon),
		HCursor:       syscall.Handle(cursor),
		LpszClassName: className,
		HIconSm:       syscall.Handle(icon),
	}
	if atom, _, err := pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc))); atom == 0 {
		messageBox("Pageant-Bruecke", "Fensterklasse konnte nicht registriert werden:\n"+fmtErr(err), mbOK|mbIconError)
		return
	}

	// Normales (unsichtbares) Fenster, kein Message-Only-Fenster:
	// die Clients suchen es mit FindWindow, das nur Top-Level-Fenster findet.
	hwnd, _, err := pCreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(utf16Ptr(pageantTitle))),
		0, 0, 0, 0, 0, 0, 0, inst, 0)
	if hwnd == 0 {
		messageBox("Pageant-Bruecke", "Fenster konnte nicht erstellt werden:\n"+fmtErr(err), mbOK|mbIconError)
		return
	}
	mainWindow = syscall.Handle(hwnd)

	addTrayIcon(mainWindow)
	logf("Bruecke gestartet, Fenster \"%s\" registriert", pageantTitle)
	if agentReachable() {
		logf("Bitwarden SSH-Agent ist erreichbar")
	} else {
		logf("Bitwarden SSH-Agent derzeit NICHT erreichbar (App starten, Tresor entsperren, SSH-Agent aktivieren)")
	}

	go func() {
		for {
			time.Sleep(30 * time.Second)
			updateTip()
		}
	}()

	var m msgStruct
	for {
		ret, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
}
