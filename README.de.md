# bwpageant

[![build](https://github.com/remiind/bwpagent/actions/workflows/build.yml/badge.svg)](https://github.com/remiind/bwpagent/actions/workflows/build.yml)
[![License: GPL v3](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)

🇬🇧 [English version](README.md)

**Pageant-Brücke zum Bitwarden SSH-Agent.** Kleine Tray-Anwendung, die ein
Pageant-Fenster bereitstellt und alle Agent-Anfragen unverändert an den SSH-Agent
der Bitwarden-Desktop-App weiterreicht.

Damit funktionieren **MobaXterm, PuTTY, WinSCP und KiTTY** mit Schlüsseln, die im
Bitwarden- oder Vaultwarden-Tresor liegen. Auf der Festplatte liegt kein Private Key.

```
MobaXterm  --WM_COPYDATA + Shared Memory-->  bwpageant.exe  --Named Pipe-->  Bitwarden-Agent
```

Der Broker sieht nur öffentliche Schlüssel und Signaturanfragen. Private Keys
verlassen den Tresor nie, jede Signatur bestätigst du weiterhin in Bitwarden.

## Voraussetzungen

* Windows 10/11 (x64)
* Bitwarden-Desktop-App, Tresor entsperrt, **SSH-Agent aktiviert**
* Windows-Dienst `OpenSSH Authentication Agent` **deaktiviert**
  (sonst belegt er die Named Pipe):

  ```powershell
  Stop-Service ssh-agent
  Set-Service ssh-agent -StartupType Disabled
  ```

* Pageant selbst darf **nicht** laufen. Es kann nur ein Fenster namens `Pageant` geben.

## Installation

1. `bwpageant.exe` von der [Releases](https://github.com/remiind/bwpagent/releases)-Seite
   laden (oder [selbst bauen](#selbst-bauen)) und an einen festen Ort legen,
   z. B. `C:\Tools\bwpageant\bwpageant.exe`.
2. Starten. Es erscheint ein Tray-Symbol.
3. Rechtsklick auf das Symbol → **Mit Windows starten** anhaken.

## MobaXterm einrichten

Settings → Configuration → SSH:

* **Use internal SSH agent (MobAgent)**: aus
* **Use external Pageant**: an

Danach Session öffnen. Bitwarden fragt beim ersten Zugriff nach Freigabe.
Der Bestätigungsdialog öffnet sich manchmal hinter anderen Fenstern, also auf
die Taskleiste achten, wenn die Verbindung zu hängen scheint.

PuTTY, KiTTY und WinSCP nutzen Pageant automatisch, sobald es läuft.

## Tray-Menü

| Eintrag | Bedeutung |
| --- | --- |
| Bitwarden-Agent | Zeigt, ob die Named Pipe gerade erreichbar ist |
| Bearbeitete Anfragen | Zähler seit dem Start |
| Mit Windows starten | Legt bzw. entfernt `HKCU\...\CurrentVersion\Run\BwPageantBridge` |
| Protokoll anzeigen | Schreibt `%TEMP%\bwpageant.log` und öffnet es im Editor |
| Beenden | Beendet die Brücke |

## Sicherheitsmaßnahmen im Code

* **Eigentümerprüfung**: Das Shared-Memory-Objekt der Anfrage muss demselben
  Windows-Benutzer gehören (`GetSecurityInfo` + `EqualSid`). Prozesse anderer
  Konten werden abgewiesen, genau wie beim echten Pageant.
* **Whitelist der Nachrichtentypen**: Nur `SSH_AGENTC_REQUEST_IDENTITIES` (11)
  und `SSH_AGENTC_SIGN_REQUEST` (13) werden durchgereicht. Hinzufügen, Löschen
  und Sperren von Schlüsseln wird mit `SSH_AGENT_FAILURE` beantwortet. Ein
  kompromittierter SSH-Client kann deinen Agenten also nicht manipulieren.
* **Längenprüfung**: Rahmen über 8192 Byte (`AGENT_MAX_MSGLEN`) werden verworfen,
  in beide Richtungen.
* **Namensprüfung**: Der Name des Shared-Memory-Objekts muss druckbares ASCII
  ohne Pfadtrenner sein und höchstens 260 Zeichen lang.
* **Strukturgrößen** werden zur Kompilierzeit gegen die Win32-Definitionen geprüft.
* **Protokoll ohne Nutzdaten**: Es werden nur Typ, Länge und Ergebnis geloggt,
  nie Schlüssel- oder Signaturmaterial.
* Go als speichersichere Sprache; die wenigen `unsafe`-Stellen sind die
  unvermeidlichen Übergaben von Windows-Zeigern und jeweils längengeprüft.

Keine externen Abhängigkeiten, nur die Go-Standardbibliothek.

## Selbst bauen

Mit installiertem Go unter Windows:

```cmd
build.cmd
```

Oder plattformunabhängig:

```bash
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-H windowsgui -s -w" -o bwpageant.exe .
```

## Bekannte Grenzen

* Nur der klassische Pageant-Weg über `WM_COPYDATA`. Clients, die ausschließlich
  die neuere PuTTY-Named-Pipe nutzen, werden nicht bedient.
* Während Bitwarden auf deine Bestätigung wartet, reagiert das Tray-Menü nicht.
  Das liegt daran, dass die Anfrage synchron beantwortet werden muss.
* MobaXterm fragt pro Sitzung teils mehrfach (Terminal und SFTP-Browser). Wie oft
  Bitwarden nachfragt, stellst du in der Bitwarden-App ein.

## Mitmachen

Issues und Pull Requests sind willkommen. Bitte keine externen Abhängigkeiten
hinzufügen und die oben genannten Sicherheitseigenschaften erhalten.

## Haftungsausschluss

Nicht mit Bitwarden oder dem PuTTY-Projekt verbunden. Nutzung auf eigene Gefahr.

## Lizenz

[GPL-3.0](LICENSE)
