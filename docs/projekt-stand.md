## Checklist

### Init
- [x] Repo angelegt (öffentlich, MIT, Copyright AgentNik007)
- [x] Ordnerstruktur angelegt
- [x] `go.mod` initialisiert (`github.com/AgentGG00/bot-core`)
- [x] Branches `dev`/`main` eingerichtet

### Backend
- [x] `core/telegram` – Anbindung, Chat-ID-Whitelist, Long Polling
- [x] `core/auth` – TOTP-Login, 6-Monats-Session, Replay-Schutz, Step-up-Funktion, `/logout`
- [x] `core/lockdown` – Lockdown-Flag lesen (dateibasiert, `state/lockdown.flag`)
- [x] `core/mask` – Maskierung von `.env`-Werten und `Bearer`-Tokens in ausgehenden Nachrichten
- [x] `core/state` – SQLite-Init für Sessions/Replay-Schutz

### Framework
- [x] SQLite-Schema für Sessions und TOTP-Replay-Schutz (Lockdown bewusst dateibasiert, nicht in SQLite)

### Features
#### feat: Auth & Sicherheit
- [x] TOTP-Login mit Session (6 Monate, Replay-Schutz)
- [x] Step-up-TOTP-Primitive (`auth.VerifyStepUp`) – Bindung an konkrete Aktionen ist Sache der jeweiligen Anwendung im Source-Repo
- [x] Lockdown-Flag-Check als wiederverwendbare Funktion

### Fix
- [ ]

### Install
- [ ] `scripts/lockdown.sh`, `scripts/unlock.sh` auf dem Server ausführbar hinterlegt

### Test / Review
- [ ] Als Go-Module von einem Source-Repo eingebunden und end-to-end getestet

### Deployment
- [ ] Erstes stabiles Tag/Release erstellt, das Source-Repos als Dependency referenzieren können