# bot-core

[![Version](https://img.shields.io/github/v/release/AgentGG00/bot-core)](https://github.com/AgentGG00/bot-core/releases)
[![Status](https://img.shields.io/badge/status-WIP-yellow)]()
[![License](https://img.shields.io/github/license/AgentGG00/bot-core)](https://github.com/AgentGG00/bot-core/blob/main/LICENSE)

Wiederverwendbarer Telegram-Steuerungs-Core in Go. Übernimmt Telegram-Anbindung, TOTP-Login,
Lockdown, Scheduler-Mechanik und lädt eine Source, die definiert, was der Bot konkret tut.

> Läuft **nicht eigenständig**. `bot-core` braucht immer eine Source (siehe
> [bot-source-template](https://github.com/AgentGG00/bot-source-template)) – ohne Source
> hat der Bot keine `.env`, keine Konfiguration und keine Aufgabe.

## Features
- Telegram-Bot mit Chat-ID-Whitelist
- TOTP-Login (RFC 6238), 6-Monats-Session, Step-up-Bestätigung bei riskanten Aktionen
- Lockdown per SSH-Skript (`scripts/lockdown.sh` / `scripts/unlock.sh`), unabhängig vom Chat
- Scheduler: dauerhafte Jobs, Sperrfenster mit Nachlauf, Bedingungsskripte (übersteht Neustarts)
- Automatische Maskierung sensibler Daten in ausgehenden Nachrichten
- Optionaler LLM-Provider (Ollama) für Zusammenfassungen, mit Fallback auf unbearbeiteten Text

## Architecture
- **Core** (dieses Repo): Mechanik – Telegram, Auth, Lockdown, Scheduler, Source-Loader, LLM-Client
- **Source** (privates Repo pro Anwendung, aus `bot-source-template`): Policy – was wird geprüft,
  welche Skripte laufen, welche Sperrfenster gelten, welche Buttons es gibt

## Requirements
- Go 1.23+
- SQLite (via `modernc.org/sqlite`, kein separater Server nötig)
- Eine Source nach dem Schema von `bot-source-template`

## Usage
```bash
bot --source /pfad/zur/source
```
Der Core liest `.env`, `config/source.yaml`, `config/prompt.md` und `config/context.md` aus dem
Source-Verzeichnis und legt seinen Zustand (Jobs, Sessions, Lockdown) als SQLite-Datei dort ab.

## Local Development
1. Repo klonen
2. Eine Source lokal bereitstellen (z. B. `bot-source-template` mit ausgefüllten Werten)
3. `go build ./src/cmd/bot`
4. `./bot --source /pfad/zur/source`

## Security
- TOTP schützt Login und riskante Aktionen (Rollback, Restore, Löschen, Flag entfernen)
- Lockdown läuft ausschließlich über SSH (`scripts/lockdown.sh`), nicht über Telegram
- Ausgehende Nachrichten werden vor dem Versand auf bekannte Secret-Muster geprüft und maskiert
- Sicherheitslücken bitte über GitHub Issues melden

## License
MIT, siehe [LICENSE](./LICENSE)