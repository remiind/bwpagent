//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")

	pRegisterClassEx       = user32.NewProc("RegisterClassExW")
	pCreateWindowEx        = user32.NewProc("CreateWindowExW")
	pDefWindowProc         = user32.NewProc("DefWindowProcW")
	pGetMessage            = user32.NewProc("GetMessageW")
	pTranslateMessage      = user32.NewProc("TranslateMessage")
	pDispatchMessage       = user32.NewProc("DispatchMessageW")
	pPostQuitMessage       = user32.NewProc("PostQuitMessage")
	pDestroyWindow         = user32.NewProc("DestroyWindow")
	pFindWindow            = user32.NewProc("FindWindowW")
	pLoadIcon              = user32.NewProc("LoadIconW")
	pLoadCursor            = user32.NewProc("LoadCursorW")
	pCreatePopupMenu       = user32.NewProc("CreatePopupMenu")
	pAppendMenu            = user32.NewProc("AppendMenuW")
	pTrackPopupMenu        = user32.NewProc("TrackPopupMenu")
	pDestroyMenu           = user32.NewProc("DestroyMenu")
	pSetForegroundWin      = user32.NewProc("SetForegroundWindow")
	pGetCursorPos          = user32.NewProc("GetCursorPos")
	pMessageBox            = user32.NewProc("MessageBoxW")
	pPostMessage           = user32.NewProc("PostMessageW")
	pGetModuleHandle       = kernel32.NewProc("GetModuleHandleW")
	pOpenFileMapping       = kernel32.NewProc("OpenFileMappingW")
	pMapViewOfFile         = kernel32.NewProc("MapViewOfFile")
	pUnmapViewOfFile       = kernel32.NewProc("UnmapViewOfFile")
	pCloseHandle           = kernel32.NewProc("CloseHandle")
	pVirtualQuery          = kernel32.NewProc("VirtualQuery")
	pLocalFree             = kernel32.NewProc("LocalFree")
	pGetCurrentProcessId   = kernel32.NewProc("GetCurrentProcessId")
	pOpenProcess           = kernel32.NewProc("OpenProcess")
	pConvertSidToStringSid = advapi32.NewProc("ConvertSidToStringSidW")
	pGetSecurityInfo       = advapi32.NewProc("GetSecurityInfo")
	pEqualSid              = advapi32.NewProc("EqualSid")
	pRegCreateKeyEx        = advapi32.NewProc("RegCreateKeyExW")
	pRegSetValueEx         = advapi32.NewProc("RegSetValueExW")
	pRegDeleteValue        = advapi32.NewProc("RegDeleteValueW")
	pRegQueryValueEx       = advapi32.NewProc("RegQueryValueExW")
	pRegCloseKey           = advapi32.NewProc("RegCloseKey")
	pShellNotifyIcon       = shell32.NewProc("Shell_NotifyIconW")
	pShellExecute          = shell32.NewProc("ShellExecuteW")
)

// Fenster- und Menü-Konstanten
const (
	wmNull          = 0x0000
	wmDestroy       = 0x0002
	wmClose         = 0x0010
	wmCopyData      = 0x004A
	wmCommand       = 0x0111
	wmRButtonUp     = 0x0205
	wmLButtonDblClk = 0x0203
	wmApp           = 0x8000

	wmTrayIcon = wmApp + 1

	mfString    = 0x0000
	mfSeparator = 0x0800
	mfChecked   = 0x0008
	mfUnchecked = 0x0000

	tpmRightButton = 0x0002
	tpmBottomAlign = 0x0020

	idiApplication = 32512
	idcArrow       = 32512

	nimAdd    = 0x0000
	nimModify = 0x0001
	nimDelete = 0x0002

	nifMessage = 0x0001
	nifIcon    = 0x0002
	nifTip     = 0x0004

	mbOK        = 0x0000
	mbIconInfo  = 0x0040
	mbIconWarn  = 0x0030
	mbIconError = 0x0010

	swShow = 5
)

// Speicher- und Sicherheits-Konstanten
const (
	fileMapRead  = 0x0004
	fileMapWrite = 0x0002

	readControl          = 0x00020000
	seKernelObject       = 6
	ownerSecurityInfo    = 0x00000001
	errorSuccess         = 0
	hkeyCurrentUser      = 0x80000001
	keyWrite             = 0x20006
	keyRead              = 0x20019
	regSZ                = 1
	regOptionNonVolatile = 0
)

type point struct {
	X int32
	Y int32
}

type msgStruct struct {
	Hwnd    syscall.Handle
	Message uint32
	_       uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
	_       uint32
}

type wndClassEx struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     syscall.Handle
	HIcon         syscall.Handle
	HCursor       syscall.Handle
	HbrBackground syscall.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       syscall.Handle
}

// COPYDATASTRUCT
type copyDataStruct struct {
	DwData uintptr
	CbData uint32
	LpData uintptr
}

type memoryBasicInformation struct {
	BaseAddress       uintptr
	AllocationBase    uintptr
	AllocationProtect uint32
	_                 uint32
	RegionSize        uintptr
	State             uint32
	Protect           uint32
	Type              uint32
	_                 uint32
}

type notifyIconData struct {
	CbSize            uint32
	_                 uint32
	HWnd              syscall.Handle
	UID               uint32
	UFlags            uint32
	UCallbackMessage  uint32
	_                 uint32
	HIcon             syscall.Handle
	SzTip             [128]uint16
	DwState           uint32
	DwStateMask       uint32
	SzInfo            [256]uint16
	UTimeoutOrVersion uint32
	SzInfoTitle       [64]uint16
	DwInfoFlags       uint32
	GuidItem          [16]byte
	HBalloonIcon      syscall.Handle
}

// Kompilierzeit-Pruefung der Strukturgroessen gegen die Win32-Definitionen
// fuer x64. Stimmt ein Wert nicht, schlaegt der Build fehl, statt dass zur
// Laufzeit falsche Speicherbereiche gelesen werden.
const (
	_ = uint64(unsafe.Sizeof(wndClassEx{})) - 80
	_ = 80 - uint64(unsafe.Sizeof(wndClassEx{}))
	_ = uint64(unsafe.Sizeof(msgStruct{})) - 48
	_ = 48 - uint64(unsafe.Sizeof(msgStruct{}))
	_ = uint64(unsafe.Sizeof(copyDataStruct{})) - 24
	_ = 24 - uint64(unsafe.Sizeof(copyDataStruct{}))
	_ = uint64(unsafe.Sizeof(memoryBasicInformation{})) - 48
	_ = 48 - uint64(unsafe.Sizeof(memoryBasicInformation{}))
	_ = uint64(unsafe.Sizeof(notifyIconData{})) - 976
	_ = 976 - uint64(unsafe.Sizeof(notifyIconData{}))
)

func utf16Ptr(s string) *uint16 {
	p, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		return nil
	}
	return p
}

func messageBox(title, text string, flags uintptr) {
	t := utf16Ptr(text)
	c := utf16Ptr(title)
	pMessageBox.Call(0,
		uintptr(unsafe.Pointer(t)),
		uintptr(unsafe.Pointer(c)),
		flags)
}
