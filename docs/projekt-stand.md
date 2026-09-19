## Checklist

### Init
- [x] Repo angelegt (öffentlich, MIT, Copyright AgentNik007)
- [x] Ordnerstruktur angelegt
- [ ] `go.mod` initialisiert (`github.com/AgentGG00/bot-core`)
- [ ] Branches `dev`/`main` eingerichtet

### Backend
- [ ] `core/telegram` – Anbindung, Chat-ID-Whitelist, Long Polling
- [ ] `core/auth` – TOTP-Login, 6-Monats-Session, Step-up für riskante Aktionen, `/logout`
- [ ] `core/lockdown` – Lockdown-Flag lesen/schreiben, wird von `scripts/lockdown.sh` gesetzt
- [ ] `core/scheduler` – dauerhafte Jobs (SQLite), Sperrfenster inkl. Nachlauf, Bedingungsskripte
- [ ] `core/sourceloader` – liest `source.yaml`, `prompt.md`, `context.md` aus dem Source-Pfad
- [ ] `core/llm` – Ollama-Client (Map-Reduce-Zusammenfassung), Fallback bei Nichterreichbarkeit
- [ ] `core/mask` – Maskierung von `.env`-Werten und bekannten Secret-Mustern in Nachrichten

### Framework
- [ ] SQLite-Schema für Jobs, Sessions, Lockdown-Historie, installierte Versionen

### Features
#### feat: Auth & Sicherheit
- [ ] TOTP-Login mit Session
- [ ] Step-up-TOTP für Rollback, Restore, Löschen, Flag-Entfernen
- [ ] Lockdown per SSH-Skript, fail-closed nach Neustart

#### feat: Scheduler
- [ ] Job-Zustandsautomat (geplant → wartet → läuft → fertig/Fehler)
- [ ] Sperrfenster mit Start/Ende/Nachlauf aus `source.yaml`
- [ ] Bedingungsskripte (Exit-Code 0/1) für dynamische Sperren

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