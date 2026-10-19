# bwpageant

[![build](https://github.com/remiind/bwpagent/actions/workflows/build.yml/badge.svg)](https://github.com/remiind/bwpagent/actions/workflows/build.yml)
[![License: GPL v3](https://img.shields.io/badge/license-GPLv3-blue.svg)](LICENSE)

🇩🇪 [Deutsche Version](README.de.md)

**A Pageant bridge for the Bitwarden SSH agent.** A tiny Windows tray app that
pretends to be Pageant and forwards every request to the SSH agent built into the
Bitwarden desktop app.

This lets **MobaXterm, PuTTY, WinSCP and KiTTY** use SSH keys stored in your
Bitwarden / Vaultwarden vault. No private key ever touches the disk.

```
MobaXterm  --WM_COPYDATA + shared memory-->  bwpageant.exe  --named pipe-->  Bitwarden agent
```

The bridge only sees public keys and signing requests. Private keys never leave
the vault, and you still approve each signature in Bitwarden.

## Why?

Bitwarden's SSH agent speaks the OpenSSH named-pipe protocol
(`\\.\pipe\openssh-ssh-agent`). Many classic Windows tools only know the Pageant
protocol (a hidden window plus `WM_COPYDATA` and shared memory). This tool
translates between the two.

## Requirements

* Windows 10/11 (x64)
* Bitwarden desktop app, vault unlocked, **SSH agent enabled**
  (Settings → SSH agent)
* The Windows service `OpenSSH Authentication Agent` **disabled**, otherwise it
  occupies the named pipe:

  ```powershell
  Stop-Service ssh-agent
  Set-Service ssh-agent -StartupType Disabled
  ```

* Real Pageant must **not** be running. Only one window named `Pageant` can exist.

## Installation

1. Download `bwpageant.exe` from the [Releases](https://github.com/remiind/bwpagent/releases)
   page (or [build it yourself](#building)) and put it somewhere permanent,
   e.g. `C:\Tools\bwpageant\bwpageant.exe`.
2. Start it. A tray icon appears.
3. Right-click the icon → tick **Mit Windows starten** (start with Windows).

## MobaXterm setup

Settings → Configuration → SSH:

* **Use internal SSH agent (MobAgent)**: off
* **Use external Pageant**: on

Open a session. Bitwarden asks for approval on first use. The confirmation dialog
sometimes opens behind other windows, so check the taskbar if the connection seems
to hang.

PuTTY, KiTTY and WinSCP use Pageant automatically when it is running.

## Tray menu

The menu is currently in German.

| Entry | Meaning |
| --- | --- |
| Bitwarden-Agent | Shows whether the named pipe is currently reachable |
| Bearbeitete Anfragen | Request counter since start |
| Mit Windows starten | Creates / removes `HKCU\...\CurrentVersion\Run\BwPageantBridge` |
| Protokoll anzeigen | Writes `%TEMP%\bwpageant.log` and opens it in the editor |
| Beenden | Quits the bridge |

## Security measures

* **Owner check**: the shared-memory object of a request must belong to the same
  Windows user (`GetSecurityInfo` + `EqualSid`), just like real Pageant. Processes
  of other accounts are rejected.
* **Message whitelist**: only `SSH_AGENTC_REQUEST_IDENTITIES` (11) and
  `SSH_AGENTC_SIGN_REQUEST` (13) are forwarded. Adding, removing and locking keys is
  answered with `SSH_AGENT_FAILURE`, so a compromised SSH client cannot tamper with
  your agent.
* **Length checks**: frames larger than 8192 bytes (`AGENT_MAX_MSGLEN`) are
  dropped, in both directions.
* **Name validation**: the shared-memory name must be printable ASCII without path
  separators and at most 260 characters.
* **Struct sizes** are checked against the Win32 definitions at compile time.
* **Log without payloads**: only message type, length and result are logged, never
  key or signature material.
* Written in Go. The few `unsafe` spots are the unavoidable handovers of Windows
  pointers and are length-checked.

No external dependencies, only the Go standard library.

## Building

With Go installed (on Windows):

```cmd
build.cmd
```

Or cross-platform:

```bash
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-H windowsgui -s -w" -o bwpageant.exe .
```

## Known limitations

* Only the classic Pageant mechanism (`WM_COPYDATA`). Clients that use only the
  newer PuTTY named-pipe mechanism are not served.
* While Bitwarden waits for your confirmation, the tray menu is unresponsive,
  because the request has to be answered synchronously.
* MobaXterm may ask several times per session (terminal and SFTP browser). How often
  Bitwarden prompts is configured in the Bitwarden app.
* The UI strings and code comments are in German. Translations welcome.

## Contributing

Issues and pull requests are welcome. Please keep the dependency list empty and the
security properties above intact.

## Disclaimer

Not affiliated with Bitwarden or the PuTTY project. Use at your own risk.

## License

[GPL-3.0](LICENSE)
