//go:build windows

package main

import (
	"encoding/binary"
	"fmt"
	"syscall"
	"unsafe"
)

const (
	// Kennung, mit der PuTTY/MobaXterm eine Agent-Anfrage per WM_COPYDATA schickt.
	agentCopyDataID = 0x804e50ba
	// Maximale Nachrichtengroesse des Pageant-Protokolls.
	agentMaxMsgLen = 8192
	// Laengstmoeglicher Name eines Shared-Memory-Objekts, den wir akzeptieren.
	maxMapNameLen = 260
)

// handleCopyData verarbeitet eine Pageant-Anfrage.
// Rueckgabe 1 = bearbeitet, 0 = abgelehnt (Client meldet dann Fehler).
func handleCopyData(lParam uintptr) uintptr {
	cds := (*copyDataStruct)(unsafe.Pointer(lParam))
	if cds.DwData != agentCopyDataID {
		return 0
	}
	name, ok := readMapName(cds)
	if !ok {
		logf("Anfrage abgelehnt: ungueltiger Name des Shared-Memory-Objekts")
		return 0
	}

	hMap, _, err := pOpenFileMapping.Call(
		uintptr(fileMapRead|fileMapWrite), 0,
		uintptr(unsafe.Pointer(utf16Ptr(name))))
	if hMap == 0 {
		logf("Anfrage abgelehnt: Shared Memory nicht zu oeffnen (%v)", err)
		return 0
	}
	defer pCloseHandle.Call(hMap)

	if !ownedByCurrentUser(syscall.Handle(hMap)) {
		logf("Anfrage ABGELEHNT: Shared Memory gehoert einem anderen Benutzer")
		return 0
	}

	view, _, err := pMapViewOfFile.Call(hMap, uintptr(fileMapRead|fileMapWrite), 0, 0, 0)
	if view == 0 {
		logf("Anfrage abgelehnt: MapViewOfFile fehlgeschlagen (%v)", err)
		return 0
	}
	defer pUnmapViewOfFile.Call(view)

	size := regionSize(view)
	if size < 5 {
		logf("Anfrage abgelehnt: Shared Memory zu klein (%d Byte)", size)
		return 0
	}
	if size > agentMaxMsgLen {
		size = agentMaxMsgLen
	}
	buf := unsafe.Slice((*byte)(unsafe.Pointer(view)), size)

	reqLen := binary.BigEndian.Uint32(buf[0:4])
	if reqLen == 0 || uint64(reqLen)+4 > uint64(size) {
		logf("Anfrage abgelehnt: unplausible Laenge (%d Byte)", reqLen)
		return 0
	}
	// Eigene Kopie, damit die weitere Verarbeitung nicht auf fremdem Speicher arbeitet,
	// den der aufrufende Prozess nebenher aendern koennte.
	req := make([]byte, 4+int(reqLen))
	copy(req, buf[:4+int(reqLen)])

	resp := serveAgentRequest(req)
	if 4+len(resp) > int(size) {
		logf("Antwort verworfen: passt nicht in den Puffer (%d Byte)", len(resp))
		resp = failureResponse()
	}
	binary.BigEndian.PutUint32(buf[0:4], uint32(len(resp)))
	copy(buf[4:], resp)
	return 1
}

// readMapName liest den Namen des Shared-Memory-Objekts aus der COPYDATASTRUCT
// und prueft ihn streng, bevor er an OpenFileMapping geht.
func readMapName(cds *copyDataStruct) (string, bool) {
	if cds.LpData == 0 || cds.CbData == 0 || cds.CbData > maxMapNameLen {
		return "", false
	}
	raw := unsafe.Slice((*byte)(unsafe.Pointer(cds.LpData)), int(cds.CbData))
	if raw[len(raw)-1] != 0 {
		return "", false
	}
	body := raw[:len(raw)-1]
	if len(body) == 0 {
		return "", false
	}
	for _, c := range body {
		// Nur unauffaelliges ASCII zulassen: keine Pfadtrenner, keine Steuerzeichen.
		if c < 0x21 || c > 0x7e || c == '\\' || c == '/' {
			return "", false
		}
	}
	return string(body), true
}

// objectOwner liefert die Eigentuemer-SID eines Kernel-Objekts sowie den
// Security Descriptor, der danach freigegeben werden muss.
func objectOwner(h uintptr) (sid uintptr, sd uintptr, ok bool) {
	ret, _, _ := pGetSecurityInfo.Call(
		h, seKernelObject, ownerSecurityInfo,
		uintptr(unsafe.Pointer(&sid)), 0, 0, 0,
		uintptr(unsafe.Pointer(&sd)))
	if ret != errorSuccess || sid == 0 {
		return 0, 0, false
	}
	return sid, sd, true
}

// ownDefaultOwner ermittelt, wem Kernel-Objekte gehoeren, die dieser Prozess
// anlegt. Bei Mitgliedern der Administratorengruppe ist das nicht der Benutzer,
// sondern die Gruppe "Administratoren". Genau so prueft auch Pageant.
func ownDefaultOwner() (sid uintptr, sd uintptr, ok bool) {
	pid, _, _ := pGetCurrentProcessId.Call()
	h, _, _ := pOpenProcess.Call(readControl, 0, pid)
	if h == 0 {
		return 0, 0, false
	}
	defer pCloseHandle.Call(h)
	return objectOwner(h)
}

func sameSid(a, b uintptr) bool {
	if a == 0 || b == 0 {
		return false
	}
	eq, _, _ := pEqualSid.Call(a, b)
	return eq != 0
}

// sidString wandelt eine SID in die uebliche S-1-5-… Schreibweise, nur fuers Protokoll.
func sidString(sid uintptr) string {
	if sid == 0 {
		return "(keine)"
	}
	var str uintptr
	ret, _, _ := pConvertSidToStringSid.Call(sid, uintptr(unsafe.Pointer(&str)))
	if ret == 0 || str == 0 {
		return "(unlesbar)"
	}
	defer pLocalFree.Call(str)
	return syscall.UTF16ToString(unsafe.Slice((*uint16)(unsafe.Pointer(str)), 256))
}

// ownedByCurrentUser prueft, ob das Shared-Memory-Objekt der Anfrage demselben
// Sicherheitskontext gehoert wie dieser Prozess. So kann kein Prozess eines
// anderen Kontos den Broker nutzen.
func ownedByCurrentUser(h syscall.Handle) bool {
	owner, sd, ok := objectOwner(uintptr(h))
	if !ok {
		logf("Eigentuemer des Shared Memory nicht lesbar")
		return false
	}
	defer pLocalFree.Call(sd)

	// 1. Standard-Eigentuemer unserer eigenen Objekte (so macht es Pageant).
	mine, mineSd, ok := ownDefaultOwner()
	if ok {
		defer pLocalFree.Call(mineSd)
		if sameSid(owner, mine) {
			return true
		}
	}

	// 2. Benutzer-SID aus dem eigenen Token, falls der Standard-Eigentuemer abweicht.
	var userSid uintptr
	if me, err := currentUserSid(); err == nil {
		userSid = uintptr(unsafe.Pointer(me))
		if sameSid(owner, userSid) {
			return true
		}
	}

	logf("Eigentuemer passt nicht: Anfrage %s, eigene Objekte %s, eigener Benutzer %s",
		sidString(owner), sidString(mine), sidString(userSid))
	return false
}

func currentUserSid() (*syscall.SID, error) {
	tok, err := syscall.OpenCurrentProcessToken()
	if err != nil {
		return nil, err
	}
	defer tok.Close()
	u, err := tok.GetTokenUser()
	if err != nil {
		return nil, err
	}
	return u.User.Sid, nil
}

func regionSize(addr uintptr) int {
	var mbi memoryBasicInformation
	ret, _, _ := pVirtualQuery.Call(addr,
		uintptr(unsafe.Pointer(&mbi)), unsafe.Sizeof(mbi))
	if ret == 0 {
		return 0
	}
	if mbi.RegionSize > agentMaxMsgLen {
		return agentMaxMsgLen
	}
	return int(mbi.RegionSize)
}

func fmtErr(err error) string {
	if err == nil {
		return "kein Fehler"
	}
	return fmt.Sprintf("%v", err)
}
