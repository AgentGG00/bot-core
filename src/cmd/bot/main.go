package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AgentGG00/bot-core/src/core/auth"
	"github.com/AgentGG00/bot-core/src/core/github"
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
	ghClient := github.New()
	sched := scheduler.New(db, src)
	pending := newPendingStore()

	log.Printf("bot: started, source=%s", src.Dir)

	go runSchedulerLoop(db, src, sched, bot)
	go runReleasePollLoop(db, src, ghClient, llmClient, sched, bot)

	for upd := range bot.Updates() {
		if lockdown.IsLocked(src.Dir) {
			log.Println("bot: lockdown active, ignoring update")
			continue
		}
		handleUpdate(db, totpSecret, bot, sched, pending, upd)
	}
}

// pendingCustom tracks a chat that clicked "Uhrzeit wählen" and is expected
// to reply with a time next.
type pendingCustom struct {
	Target  string
	Version string
}

type pendingStore struct {
	mu    sync.Mutex
	items map[int64]pendingCustom
}

func newPendingStore() *pendingStore {
	return &pendingStore{items: make(map[int64]pendingCustom)}
}

func (p *pendingStore) set(chatID int64, v pendingCustom) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.items[chatID] = v
}

func (p *pendingStore) take(chatID int64) (pendingCustom, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	v, ok := p.items[chatID]
	if ok {
		delete(p.items, chatID)
	}
	return v, ok
}

func handleUpdate(db *sql.DB, totpSecret string, bot *telegram.Bot, sched *scheduler.Scheduler, pending *pendingStore, upd telegram.Update) {
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

	if upd.CallbackData != "" {
		handleCallback(bot, sched, pending, upd)
		return
	}

	if p, ok := pending.take(upd.ChatID); ok {
		handleCustomTime(bot, sched, p, upd.Text)
		return
	}

	switch upd.Text {
	case "logout":
		if err := auth.Logout(db, upd.ChatID); err != nil {
			log.Printf("bot: logout failed: %v", err)
			return
		}
		bot.Send("Abgemeldet.")
	default:
		// TODO: weitere Befehle (manuelle Backups/Restore/Delete, Status, etc.)
		log.Printf("bot: received message %q (not yet implemented)", upd.Text)
	}
}

func handleCallback(bot *telegram.Bot, sched *scheduler.Scheduler, pending *pendingStore, upd telegram.Update) {
	if target, version, ok := parseSwitchCallback(upd.CallbackData); ok {
		handleSwitch(bot, sched, target, version)
		return
	}

	target, version, action, ok := parseUpdateCallback(upd.CallbackData)
	if !ok {
		log.Printf("bot: received unknown callback %q", upd.CallbackData)
		return
	}

	switch action {
	case "skip":
		bot.Send(fmt.Sprintf("%s %s übersprungen.", target, version))

	case "now":
		scheduleJob(bot, sched, target, version, time.Now())

	case "tonight":
		scheduleJob(bot, sched, target, version, nextThreeAM())

	case "custom":
		pending.set(upd.ChatID, pendingCustom{Target: target, Version: version})
		bot.Send(fmt.Sprintf("Uhrzeit für %s %s bitte im Format HH:MM senden.", target, version))

	default:
		log.Printf("bot: unknown update action %q", action)
	}
}

func handleCustomTime(bot *telegram.Bot, sched *scheduler.Scheduler, p pendingCustom, text string) {
	t, err := time.Parse("15:04", strings.TrimSpace(text))
	if err != nil {
		bot.Send("Ungültiges Format, bitte HH:MM senden (z.B. 22:30).")
		return
	}

	now := time.Now()
	scheduledAt := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
	if scheduledAt.Before(now) {
		scheduledAt = scheduledAt.Add(24 * time.Hour)
	}

	scheduleJob(bot, sched, p.Target, p.Version, scheduledAt)
}

func handleSwitch(bot *telegram.Bot, sched *scheduler.Scheduler, target, version string) {
	activeJob, err := sched.ActiveJobForTarget(target)
	if err != nil {
		log.Printf("scheduler: failed to look up active job: %v", err)
		return
	}
	if activeJob == nil {
		bot.Send(fmt.Sprintf("Für %s ist kein Update mehr geplant.", target))
		return
	}
	if err := sched.CancelJob(activeJob.ID); err != nil {
		log.Printf("scheduler: failed to cancel job %d: %v", activeJob.ID, err)
		bot.Send(fmt.Sprintf("Konnte geplantes Update für %s nicht ersetzen.", target))
		return
	}
	scheduleJob(bot, sched, target, version, activeJob.ScheduledAt)
}

// scheduleJob shifts "at" past any currently blocking lock window, then
// creates the job. No-chain-scheduling is enforced by scheduler.CreateJob.
func scheduleJob(bot *telegram.Bot, sched *scheduler.Scheduler, target, version string, at time.Time) {
	status, err := sched.IsLocked(at)
	if err != nil {
		log.Printf("scheduler: lock window check failed: %v", err)
	} else if status.Locked && !status.FreeAt.IsZero() {
		at = status.FreeAt
	}

	job, err := sched.CreateJob(target, version, at)
	if err != nil {
		if errors.Is(err, scheduler.ErrJobAlreadyScheduled) {
			bot.Send(fmt.Sprintf("Für %s ist bereits ein Update geplant – keine Verkettung möglich.", target))
			return
		}
		log.Printf("scheduler: failed to create job: %v", err)
		bot.Send(fmt.Sprintf("Konnte Update für %s nicht planen: %v", target, err))
		return
	}

	bot.Send(fmt.Sprintf("%s %s geplant für %s.", target, version, job.ScheduledAt.Format("02.01.2006 15:04")))
}

func nextThreeAM() time.Time {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}

func parseUpdateCallback(data string) (target, version, action string, ok bool) {
	parts := strings.SplitN(data, ":", 4)
	if len(parts) != 4 || parts[0] != "update" {
		return "", "", "", false
	}
	return parts[1], parts[2], parts[3], true
}

func parseSwitchCallback(data string) (target, version string, ok bool) {
	parts := strings.SplitN(data, ":", 3)
	if len(parts) != 3 || parts[0] != "switch" {
		return "", "", false
	}
	return parts[1], parts[2], true
}

func runSchedulerLoop(db *sql.DB, src *sourceloader.Source, sched *scheduler.Scheduler, bot *telegram.Bot) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		due, err := sched.DueJobs(time.Now())
		if err != nil {
			log.Printf("scheduler: failed to fetch due jobs: %v", err)
			continue
		}
		for _, job := range due {
			runJob(db, src, sched, bot, job)
		}
	}
}

func runJob(db *sql.DB, src *sourceloader.Source, sched *scheduler.Scheduler, bot *telegram.Bot, job *scheduler.Job) {
	status, err := sched.IsLocked(time.Now())
	if err != nil {
		log.Printf("scheduler: lock window check failed for job %d: %v", job.ID, err)
		return
	}
	if status.Locked {
		retry := status.FreeAt
		if retry.IsZero() || !retry.After(time.Now()) {
			retry = time.Now().Add(15 * time.Minute)
		}
		if err := sched.Reschedule(job.ID, retry); err != nil {
			log.Printf("scheduler: failed to reschedule job %d: %v", job.ID, err)
		}
		return
	}

	target, ok := findTarget(src, job.Target)
	if !ok {
		log.Printf("scheduler: job %d references unknown target %q", job.ID, job.Target)
		sched.UpdateStatus(job.ID, scheduler.StatusFailed)
		return
	}

	if target.Scripts.Update == "" {
		log.Printf("scheduler: target %q has no update command configured", job.Target)
		sched.UpdateStatus(job.ID, scheduler.StatusFailed)
		return
	}

	sched.UpdateStatus(job.ID, scheduler.StatusRunning)
	bot.Send(fmt.Sprintf("▶️ %s %s wird jetzt ausgeführt.", job.Target, job.Version))

	cmd := exec.Command("sh", "-c", target.Scripts.Update)
	output, err := cmd.CombinedOutput()

	if err != nil {
		sched.UpdateStatus(job.ID, scheduler.StatusFailed)
		bot.Send(fmt.Sprintf("❌ %s %s fehlgeschlagen: %v\n\n%s", job.Target, job.Version, err, string(output)))
		return
	}

	sched.UpdateStatus(job.ID, scheduler.StatusSuccess)
	if err := state.SetCurrentVersion(db, job.Target, job.Version); err != nil {
		log.Printf("scheduler: failed to record installed version for %s: %v", job.Target, err)
	}
	bot.Send(fmt.Sprintf("✅ %s erfolgreich auf %s aktualisiert.", job.Target, job.Version))
}

func findTarget(src *sourceloader.Source, name string) (sourceloader.Target, bool) {
	for _, t := range src.Config.Targets {
		if t.Name == name {
			return t, true
		}
	}
	return sourceloader.Target{}, false
}

// runReleasePollLoop periodically checks every configured Target's GitHub
// repo for a new stable release and notifies via Telegram when one appears.
func runReleasePollLoop(db *sql.DB, src *sourceloader.Source, gh *github.Client, llmClient *llm.Client, sched *scheduler.Scheduler, bot *telegram.Bot) {
	interval := time.Duration(src.Config.PollIntervalHours) * time.Hour
	if interval <= 0 {
		interval = 5 * time.Hour
	}

	checkAllTargets(db, src, gh, llmClient, sched, bot)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		checkAllTargets(db, src, gh, llmClient, sched, bot)
	}
}

func checkAllTargets(db *sql.DB, src *sourceloader.Source, gh *github.Client, llmClient *llm.Client, sched *scheduler.Scheduler, bot *telegram.Bot) {
	for _, target := range src.Config.Targets {
		if err := checkTarget(db, src, target, gh, llmClient, sched, bot); err != nil {
			log.Printf("github-poll: %s: %v", target.Name, err)
		}
	}
}

func checkTarget(db *sql.DB, src *sourceloader.Source, target sourceloader.Target, gh *github.Client, llmClient *llm.Client, sched *scheduler.Scheduler, bot *telegram.Bot) error {
	release, err := gh.LatestStable(target.GithubRepo)
	if err != nil {
		return fmt.Errorf("fetch latest release: %w", err)
	}
	if release == nil {
		return nil
	}

	known, found, err := state.GetCurrentVersion(db, target.Name)
	if err != nil {
		return fmt.Errorf("read known version: %w", err)
	}

	if found && known == release.TagName {
		return nil
	}

	if !found {
		// Erster Lauf für dieses Target: Baseline setzen, keine Benachrichtigung
		// für die bereits vorhandene Version.
		return state.SetCurrentVersion(db, target.Name, release.TagName)
	}

	activeJob, err := sched.ActiveJobForTarget(target.Name)
	if err != nil {
		return fmt.Errorf("check active job: %w", err)
	}

	if activeJob != nil {
		message := fmt.Sprintf(
			"ℹ️ %s: Update auf %s ist bereits für %s geplant. Neue Version %s verfügbar.",
			target.Name, activeJob.Version, activeJob.ScheduledAt.Format("02.01.2006 15:04"), release.TagName,
		)
		buttons := []telegram.Button{
			{Label: "Übernehmen", Data: fmt.Sprintf("switch:%s:%s", target.Name, release.TagName)},
			{Label: "Überspringen", Data: fmt.Sprintf("update:%s:%s:skip", target.Name, release.TagName)},
		}
		if err := bot.SendWithButtons(message, buttons); err != nil {
			return fmt.Errorf("send conflict notification: %w", err)
		}
		return state.SetCurrentVersion(db, target.Name, release.TagName)
	}

	summary := llmClient.Summarize(src.Prompt, src.Context, release.Body, release.HTMLURL)
	message := fmt.Sprintf("🆕 %s: neues Release %s\n\n%s", target.Name, release.TagName, summary)

	buttons := []telegram.Button{
		{Label: "Jetzt", Data: fmt.Sprintf("update:%s:%s:now", target.Name, release.TagName)},
		{Label: "Heute Nacht 3:00", Data: fmt.Sprintf("update:%s:%s:tonight", target.Name, release.TagName)},
		{Label: "Uhrzeit wählen", Data: fmt.Sprintf("update:%s:%s:custom", target.Name, release.TagName)},
		{Label: "Überspringen", Data: fmt.Sprintf("update:%s:%s:skip", target.Name, release.TagName)},
	}

	if err := bot.SendWithButtons(message, buttons); err != nil {
		return fmt.Errorf("send notification: %w", err)
	}

	return state.SetCurrentVersion(db, target.Name, release.TagName)
}
