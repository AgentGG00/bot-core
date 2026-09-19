## Checklist

### Init
- [x] Repo angelegt (öffentlich, MIT, Copyright AgentNik007)
- [x] Ordnerstruktur angelegt
- [x] `go.mod` initialisiert (`github.com/AgentGG00/bot-core`)
- [x] Branches `dev`/`main` eingerichtet

### Backend
- [x] `core/telegram` – Anbindung, Chat-ID-Whitelist, Long Polling
- [x] `core/auth` – TOTP-Login, 6-Monats-Session, Replay-Schutz, Step-up-Funktion, `/logout`
- [x] `core/lockdown` – Lockdown-Flag lesen (dateibasiert, `state/lockdown.flag`), im Polling-Loop gegated
- [x] `core/scheduler` – Job-Persistenz (SQLite), Sperrfenster-Auswertung inkl. Nachlauf, Bedingungsskripte, Zustandsübergänge, Cancel/Switch
- [x] `core/sourceloader` – liest `source.yaml`, `.env`, `prompt.md`, `context.md` aus dem Source-Pfad
- [x] `core/llm` – Ollama-Client mit Map-Reduce-Zusammenfassung, Fallback bei Nichterreichbarkeit
- [x] `core/mask` – Maskierung von `.env`-Werten und `Bearer`-Tokens in ausgehenden Nachrichten
- [x] `core/github` – Client für GitHub Releases API, filtert Draft/Prerelease
- [x] `cmd/bot` – Verdrahtung aller Core-Pakete, Polling-Loops, Callback-Handling, Job-Ausführung

### Framework
- [x] SQLite-Schema für Jobs, Sessions, TOTP-Replay-Schutz, installierte Versionen (Lockdown bewusst dateibasiert, nicht in SQLite)

### Features
#### feat: Auth & Sicherheit
- [x] TOTP-Login mit Session (6 Monate, Replay-Schutz)
- [ ] Step-up-TOTP für Rollback, Restore, Löschen, Flag-Entfernen (Funktion vorhanden, wird gebunden sobald Backup/Restore-Kommandos existieren)
- [x] Lockdown per SSH-Skript, fail-closed – Bot ignoriert alle Updates solange Flag gesetzt ist

#### feat: Scheduler
- [x] Job-Zustandsautomat (scheduled → running → success/failed, plus cancelled) inkl. tatsächlicher Script-Ausführung
- [x] Sperrfenster mit Start/Ende/Puffer aus `source.yaml`, Recheck bei Job-Ausführung
- [x] Bedingungsskripte (Exit-Code 0/1) für dynamische Sperren

#### feat: LLM-Zusammenfassung
- [x] Ollama-Client mit Map-Reduce für lange Release Notes
- [x] Fallback auf Rohtext + Link bei Nichterreichbarkeit
- [x] Anbindung an GitHub-Release-Poll-Flow

#### feat: Bot-Kommunikation
- [x] 4-Button-Nachricht (Jetzt / Heute Nacht 3:00 / Uhrzeit wählen / Überspringen)
- [x] 2-Button-Nachricht bei neuer Version während geplantem Job (Übernehmen / Überspringen)
- [x] Fehlermeldungen mit maskiertem Logauszug (Script-Output über `mask.Masker`)

### Fix
- [ ]

### Install
- [ ] `scripts/lockdown.sh`, `scripts/unlock.sh` auf dem Server ausführbar hinterlegt

### Test / Review
- [ ] Gegen `bot-source-ha-compose` end-to-end getestet

### Deployment
- [ ] Binary manuell auf Homelab gebaut/kopiert (kein CI/CD)
