//go:build windows

package main

import (
	"encoding/binary"
	"io"
	"os"
	"sync"
	"sync/atomic"
)

// Named Pipe des Windows-OpenSSH-Agent-Protokolls. Diese Pipe stellt
// die Bitwarden-Desktop-App bereit, wenn ihr SSH-Agent aktiv ist.
const agentPipe = `\\.\pipe\openssh-ssh-agent`

// Erlaubte Nachrichtentypen. Alles andere wird abgelehnt, damit ein
// SSH-Client ueber den Broker keine Schluessel hinzufuegen, loeschen
// oder den Agenten sperren kann.
const (
	msgRequestIdentities = 11
	msgSignRequest       = 13
	msgFailure           = 5
)

var (
	pipeMu        sync.Mutex
	reqCount      atomic.Int64
	lastReachable atomic.Bool
)

func failureResponse() []byte {
	return []byte{msgFailure}
}

// serveAgentRequest bekommt den kompletten Rahmen (4 Byte Laenge + Nutzlast)
// und liefert die Nutzlast der Antwort zurueck.
func serveAgentRequest(frame []byte) []byte {
	if len(frame) < 5 {
		return failureResponse()
	}
	msgType := frame[4]
	if msgType != msgRequestIdentities && msgType != msgSignRequest {
		logf("Nachrichtentyp %d abgelehnt (nur Auflisten und Signieren erlaubt)", msgType)
		return failureResponse()
	}

	pipeMu.Lock()
	defer pipeMu.Unlock()

	resp, err := forward(frame)
	if err != nil {
		logf("Weiterleitung an den Agenten fehlgeschlagen: %v", err)
		return failureResponse()
	}
	reqCount.Add(1)
	switch msgType {
	case msgRequestIdentities:
		logf("Schluesselliste angefragt, Antwort %d Byte", len(resp))
	case msgSignRequest:
		logf("Signatur angefragt, Antwort %d Byte", len(resp))
	}
	return resp
}

func forward(frame []byte) ([]byte, error) {
	f, err := os.OpenFile(agentPipe, os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := f.Write(frame); err != nil {
		return nil, err
	}

	var hdr [4]byte
	if _, err := io.ReadFull(f, hdr[:]); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n == 0 || n > agentMaxMsgLen {
		return nil, errBadLength(n)
	}
	body := make([]byte, int(n))
	if _, err := io.ReadFull(f, body); err != nil {
		return nil, err
	}
	return body, nil
}

type errBadLength uint32

func (e errBadLength) Error() string {
	return "unplausible Antwortlaenge vom Agenten"
}

// agentReachable prueft, ob die Pipe der Bitwarden-App gerade existiert.
// Laeuft gerade eine Anfrage, wird der letzte bekannte Zustand gemeldet,
// damit die Pruefung eine Signatur nicht stoert.
func agentReachable() bool {
	if !pipeMu.TryLock() {
		return lastReachable.Load()
	}
	defer pipeMu.Unlock()

	f, err := os.OpenFile(agentPipe, os.O_RDWR, 0)
	if err != nil {
		lastReachable.Store(false)
		return false
	}
	f.Close()
	lastReachable.Store(true)
	return true
}
