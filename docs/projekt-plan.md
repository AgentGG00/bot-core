# Projekt-Plan – bot-core

## Kurzbeschreibung
Wiederverwendbarer Telegram-Steuerungs-Core in Go. Übernimmt alle anwendungsunabhängigen
Aufgaben (Auth, Chat, Scheduler, Lockdown) und lädt für die konkrete Aufgabe eine externe
Source (Config + Skripte).

## Ziele & Anforderungen
- Sichere Zwei-Faktor-Steuerung (Telegram-Whitelist + TOTP) für Server-Updates aus der Ferne
- Kein wiederkehrender oder laufender Kostenanteil
- Core einmal bauen, für beliebige weitere Steuerungs-Bots wiederverwendbar
- Trennung: Core = Mechanik, Source = Policy/Anwendungslogik

## Tech-Stack
| Bereich | Wahl |
|---|---|
| Sprache | Go 1.23 |
| Telegram | `go-telegram-bot-api/telegram-bot-api` |
| TOTP | `pquerna/otp` |
| Config-Parsing | `gopkg.in/yaml.v3` |
| Zustand | SQLite über `modernc.org/sqlite` (kein CGO) |
| LLM-Anbindung | eigener HTTP-Client gegen Ollama-REST-API |
| Logging | `log/slog` (Standardbibliothek) |

## Grobe Projektstruktur
bot-core/
├── src/
│ ├── cmd/bot/
│ └── core/
│ ├── telegram/ auth/ lockdown/ scheduler/ sourceloader/ llm/ mask/
├── docs/
├── scripts/ # lockdown.sh, unlock.sh
├── go.mod go.sum
├── .gitignore README.md LICENSE projekt-plan.md projekt-stand.md

## Rahmenbedingungen
- Hosting: läuft auf dem Homelab-Server (Docker), neben den zu aktualisierenden Diensten
- LLM läuft separat auf dem VDS (Ollama, Qwen3 8B), Anbindung über Tailscale
- Repo: öffentlich, MIT-Lizenz, Copyright AgentNik007
- Branches: `dev` (Entwicklung), `main` (fertige Stände)
- Kein CI/CD, kein automatisches Deployment – Build und Rollout erfolgen manuell
- Issues offen für alle, Pull Requests nur für eingeladene Collaborators

## Secrets-Übersicht
Der Core selbst besitzt keine eigenen Secrets. Alle Secrets liegen in der `.env` der jeweiligen
Source, z. B.:

TELEGRAM_BOT_TOKEN=xxxxxxx
TELEGRAM_CHAT_ID=xxxxxxx
TOTP_SECRET=xxxxxxx
OLLAMA_HOST=xxxxxxx
HA_LONG_LIVED_TOKEN=xxxxxxx