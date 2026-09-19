package main

import (
	"database/sql"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/AgentGG00/bot-core/src/core/auth"
	"github.com/AgentGG00/bot-core/src/core/llm"
	"github.com/AgentGG00/bot-core/src/core/lockdown"
	"github.com/AgentGG00/bot-core/src/core/mask"
	"github.com/AgentGG00/bot-core/src/core/scheduler"
	"github.com/AgentGG00/bot-core/src/core/sourceloader"
	"github.com/AgentGG00/bot-core/src/core/state"
	"github.com/AgentGG00/bot-core/src/core/telegram"
)

func main() {
	sourceDir := flag.String("source", "", "path to the Source directory")
	flag.Parse()
	if *sourceDir == "" {
		log.Fatal("bot: --source is required")
	}

	src, err := sourceloader.Load(*sourceDir)
	if err != nil {
		log.Fatalf("bot: failed to load source: %v", err)
	}

	token := src.Env[src.Config.Telegram.BotTokenEnv]
	if token == "" {
		log.Fatalf("bot: telegram bot token not set (%s)", src.Config.Telegram.BotTokenEnv)
	}

	chatID, err := strconv.ParseInt(src.Env[src.Config.Telegram.ChatIDEnv], 10, 64)
	if err != nil {
		log.Fatalf("bot: invalid telegram chat id: %v", err)
	}

	totpSecret := src.Env[src.Config.Totp.SecretEnv]
	if totpSecret == "" {
		log.Fatalf("bot: totp secret not set (%s)", src.Config.Totp.SecretEnv)
	}

	ollamaHost := src.Env[src.Config.LLM.OllamaHostEnv]

	stateDir := filepath.Join(src.Dir, "state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		log.Fatalf("bot: failed to create state dir: %v", err)
	}

	db, err := state.Init(filepath.Join(stateDir, "bot.db"))
	if err != nil {
		log.Fatalf("bot: failed to init state db: %v", err)
	}
	defer db.Close()

	var secrets []string
	for _, v := range src.Env {
		secrets = append(secrets, v)
	}
	masker := mask.New(secrets)

	bot, err := telegram.New(token, chatID, masker)
	if err != nil {
		log.Fatalf("bot: failed to init telegram bot: %v", err)
	}

	llmClient := llm.New(ollamaHost, src.Config.LLM.Model)
	_ = llmClient // wired for use once GitHub release polling is implemented

	sched := scheduler.New(db, src)

	log.Printf("bot: started, source=%s", src.Dir)

	go runSchedulerLoop(sched)

	for upd := range bot.Updates() {
		if lockdown.IsLocked(src.Dir) {
			log.Println("bot: lockdown active, ignoring update")
			continue
		}
		handleUpdate(db, totpSecret, bot, upd)
	}
}

func handleUpdate(db *sql.DB, totpSecret string, bot *telegram.Bot, upd telegram.Update) {
	authed, err := auth.HasActiveSession(db, upd.ChatID)
	if err != nil {
		log.Printf("bot: session check failed: %v", err)
		return
	}

	if !authed {
		if upd.Text == "" {
			bot.Send("Bitte sende deinen TOTP-Code zur Anmeldung.")
			return
		}
		if err := auth.Login(totpSecret, upd.Text, db, upd.ChatID); err != nil {
			bot.Send("Anmeldung fehlgeschlagen: ungültiger oder bereits verwendeter Code.")
			return
		}
		bot.Send("Angemeldet. Session ist 6 Monate gültig.")
		return
	}

	switch {
	case upd.CallbackData != "":
		// TODO: dispatch scheduling actions (Jetzt / Heute Nacht / Uhrzeit wählen / Überspringen)
		// once Source targets and their update scripts are wired up.
		log.Printf("bot: received callback %q (not yet implemented)", upd.CallbackData)
	case upd.Text == "logout":
		if err := auth.Logout(db, upd.ChatID); err != nil {
			log.Printf("bot: logout failed: %v", err)
			return
		}
		bot.Send("Abgemeldet.")
	default:
		// TODO: command handling (manuelle Backups/Restore/Delete, Status, etc.)
		log.Printf("bot: received message %q (not yet implemented)", upd.Text)
	}
}

func runSchedulerLoop(sched *scheduler.Scheduler) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		due, err := sched.DueJobs(time.Now())
		if err != nil {
			log.Printf("scheduler: failed to fetch due jobs: %v", err)
			continue
		}
		for _, job := range due {
			// TODO: run the target's update script once Source script execution is wired up.
			log.Printf("scheduler: job %d for %s is due (execution not yet implemented)", job.ID, job.Target)
		}
	}
}
