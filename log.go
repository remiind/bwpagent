//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const logCapacity = 300

var (
	logMu    sync.Mutex
	logLines []string
)

// logf schreibt eine Zeile in den Ringpuffer. Es werden nur Metadaten
// protokolliert (Typ, Laenge, Ergebnis), niemals Schluessel- oder Signaturdaten.
func logf(format string, args ...any) {
	line := time.Now().Format("2006-01-02 15:04:05") + "  " + fmt.Sprintf(format, args...)
	logMu.Lock()
	defer logMu.Unlock()
	logLines = append(logLines, line)
	if len(logLines) > logCapacity {
		logLines = logLines[len(logLines)-logCapacity:]
	}
}

func logText() string {
	logMu.Lock()
	defer logMu.Unlock()
	if len(logLines) == 0 {
		return "Noch keine Ereignisse."
	}
	return strings.Join(logLines, "\r\n")
}

// writeLogFile legt das Protokoll im Temp-Verzeichnis ab und gibt den Pfad zurueck.
func writeLogFile() (string, error) {
	path := filepath.Join(os.TempDir(), "bwpageant.log")
	if err := os.WriteFile(path, []byte(logText()+"\r\n"), 0o600); err != nil {
		return "", err
	}
	return path, nil
}
