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
- [x] `core/scheduler` – Job-Persistenz (SQLite), Sperrfenster-Auswertung inkl. Nachlauf, Bedingungsskripte
- [x] `core/sourceloader` – liest `source.yaml`, `.env`, `prompt.md`, `context.md` aus dem Source-Pfad
- [x] `core/llm` – Ollama-Client mit Map-Reduce-Zusammenfassung, Fallback bei Nichterreichbarkeit
- [x] `core/mask` – Maskierung von `.env`-Werten und `Bearer`-Tokens in ausgehenden Nachrichten
- [x] `cmd/bot` – Verdrahtung aller Core-Pakete, Polling-Loop mit Lockdown-/Auth-Gating

### Framework
- [x] SQLite-Schema für Jobs, Sessions, TOTP-Replay-Schutz, installierte Versionen (Lockdown bewusst dateibasiert, nicht in SQLite)

### Features
#### feat: Auth & Sicherheit
- [x] TOTP-Login mit Session (6 Monate, Replay-Schutz)
- [ ] Step-up-TOTP für Rollback, Restore, Löschen, Flag-Entfernen (Funktion vorhanden, noch an keine Aktion gebunden)
- [x] Lockdown per SSH-Skript, fail-closed – Bot ignoriert alle Updates solange Flag gesetzt ist

#### feat: Scheduler
- [ ] Job-Zustandsautomat – Datenmodell fertig (Status, CRUD, `DueJobs`), automatische Übergänge noch nicht verdrahtet
- [x] Sperrfenster mit Start/Ende/Puffer aus `source.yaml`
- [x] Bedingungsskripte (Exit-Code 0/1) für dynamische Sperren

#### feat: LLM-Zusammenfassung
- [x] Ollama-Client mit Map-Reduce für lange Release Notes
- [x] Fallback auf Rohtext + Link bei Nichterreichbarkeit
- [ ] Anbindung an tatsächlichen GitHub-Release-Poll-Flow

#### feat: [Bot-Kommunikation]
- [ ] 4-Button-Nachricht (Jetzt / Heute Nacht / Uhrzeit wählen / Überspringen)
- [ ] 2-Button-Nachricht bei neuer Version während geplantem Job (Übernehmen / Überspringen)
- [ ] Fehlermeldungen mit maskiertem Logauszug

### Fix
- [ ]

### Install
- [ ] `scripts/lockdown.sh`, `scripts/unlock.sh` auf dem Server ausführbar hinterlegt

### Test / Review
- [ ] Gegen `bot-source-ha-compose` end-to-end getestet

### Deployment
- [ ] Binary manuell auf Homelab gebaut/kopiert (kein CI/CD)