# Projekt-Plan – bot-core

## Kurzbeschreibung
Wiederverwendbare Go-Library für TOTP-gesicherte Telegram-Steuerungs-Bots. Stellt die
anwendungsunabhängigen Bausteine (Telegram-Anbindung, TOTP-Auth, Lockdown, Secret-Masking)
bereit. Jede konkrete Anwendung (z. B. Versions-Update-Management) lebt in einem eigenen
Source-Repo, das Core als Dependency einbindet und sein eigenes Binary baut.

## Ziele & Anforderungen
- Sichere Zwei-Faktor-Steuerung (Telegram-Whitelist + TOTP) als wiederverwendbarer Baustein
- Core bleibt schlank: nur was jeder darauf aufbauende Bot braucht, kein anwendungsspezifischer Code
- Kein eigenes Binary – wird als Go-Module von Source-Repos importiert
- Trennung: Core = Mechanik (Telegram, Auth, Lockdown, Masking), Source = komplette Anwendungslogik inkl. eigener Feature-Pakete

## Tech-Stack
| Bereich | Wahl |
|---|---|
| Sprache | Go 1.27 |
| Telegram | `go-telegram-bot-api/telegram-bot-api` |
| TOTP | `pquerna/otp` |
| Zustand | SQLite über `modernc.org/sqlite` (kein CGO) – nur Sessions/Replay-Schutz |
| Logging | Standardbibliothek `log` |

## Grobe Projektstruktur
```
bot-core/
├── src/
│   └── core/
│       ├── telegram/
│       ├── auth/
│       ├── lockdown/
│       ├── mask/
│       └── state/
├── docs/
├── scripts/          # lockdown.sh, unlock.sh
├── go.mod go.sum
├── .gitignore README.md LICENSE
└── docs/projekt-plan.md docs/projekt-stand.md
```

## Rahmenbedingungen
- Repo: öffentlich, MIT-Lizenz, Copyright AgentNik007
- Branches: `dev` (Entwicklung), `main` (fertige Stände)
- Kein CI/CD – Source-Repos binden Core über `go.mod` ein, kein eigenes Deployment für Core
- Issues offen für alle, Pull Requests nur für eingeladene Collaborators

## Secrets-Übersicht
Core selbst besitzt keine eigenen Secrets. Jede Anwendung, die Core einbindet, verwaltet ihre
eigenen Secrets (z. B. per `.env`):

TELEGRAM_BOT_TOKEN=xxxxxxx
TELEGRAM_CHAT_ID=xxxxxxx
TOTP_SECRET=xxxxxxx