package scheduler

import (
	"database/sql"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/AgentGG00/bot-core/src/core/sourceloader"
)

const (
	StatusScheduled        = "scheduled"
	StatusWaitingCondition = "waiting_condition"
	StatusRunning          = "running"
	StatusSuccess          = "success"
	StatusFailed           = "failed"
	StatusCancelled        = "cancelled"
)

var ErrJobAlreadyScheduled = errors.New("scheduler: an active job already exists for this target")

type Job struct {
	ID          int64
	Target      string
	Version     string
	ScheduledAt time.Time
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Scheduler struct {
	db     *sql.DB
	source *sourceloader.Source
}

func New(db *sql.DB, source *sourceloader.Source) *Scheduler {
	return &Scheduler{db: db, source: source}
}

func isActiveStatus(status string) bool {
	return status == StatusScheduled || status == StatusWaitingCondition || status == StatusRunning
}

func (s *Scheduler) hasActiveJob(target string) (bool, error) {
	row := s.db.QueryRow(
		`SELECT COUNT(*) FROM jobs WHERE target = ? AND status IN (?, ?, ?)`,
		target, StatusScheduled, StatusWaitingCondition, StatusRunning,
	)
	var count int
	if err := row.Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateJob schedules a new job for target. Fails with ErrJobAlreadyScheduled
// if an active (non-terminal) job already exists for that target — chain
// scheduling is never allowed.
func (s *Scheduler) CreateJob(target, version string, scheduledAt time.Time) (*Job, error) {
	active, err := s.hasActiveJob(target)
	if err != nil {
		return nil, err
	}
	if active {
		return nil, ErrJobAlreadyScheduled
	}

	now := time.Now().Unix()
	res, err := s.db.Exec(
		`INSERT INTO jobs (target, version, scheduled_at, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		target, version, scheduledAt.Unix(), StatusScheduled, now, now,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return s.GetJob(id)
}

func (s *Scheduler) GetJob(id int64) (*Job, error) {
	row := s.db.QueryRow(
		`SELECT id, target, version, scheduled_at, status, created_at, updated_at FROM jobs WHERE id = ?`,
		id,
	)
	return scanJob(row)
}

// ActiveJobForTarget returns the earliest active (non-terminal) job for
// target, or nil if none exists.
func (s *Scheduler) ActiveJobForTarget(target string) (*Job, error) {
	row := s.db.QueryRow(
		`SELECT id, target, version, scheduled_at, status, created_at, updated_at FROM jobs WHERE target = ? AND status IN (?, ?, ?) ORDER BY scheduled_at ASC LIMIT 1`,
		target, StatusScheduled, StatusWaitingCondition, StatusRunning,
	)
	job, err := scanJob(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (s *Scheduler) ListActiveJobs() ([]*Job, error) {
	rows, err := s.db.Query(
		`SELECT id, target, version, scheduled_at, status, created_at, updated_at FROM jobs WHERE status IN (?, ?, ?) ORDER BY scheduled_at ASC`,
		StatusScheduled, StatusWaitingCondition, StatusRunning,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job, err := scanJobRows(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (s *Scheduler) ListJobsByTarget(target string) ([]*Job, error) {
	rows, err := s.db.Query(
		`SELECT id, target, version, scheduled_at, status, created_at, updated_at FROM jobs WHERE target = ? ORDER BY scheduled_at DESC`,
		target,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job, err := scanJobRows(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (s *Scheduler) UpdateStatus(id int64, status string) error {
	_, err := s.db.Exec(
		`UPDATE jobs SET status = ?, updated_at = ? WHERE id = ?`,
		status, time.Now().Unix(), id,
	)
	return err
}

func (s *Scheduler) Reschedule(id int64, newTime time.Time) error {
	_, err := s.db.Exec(
		`UPDATE jobs SET scheduled_at = ?, updated_at = ? WHERE id = ?`,
		newTime.Unix(), time.Now().Unix(), id,
	)
	return err
}

// CancelJob marks a job as cancelled (e.g. replaced by a newer version via
// the "Übernehmen" flow). Cancelled jobs are terminal and no longer count
// as active.
func (s *Scheduler) CancelJob(id int64) error {
	return s.UpdateStatus(id, StatusCancelled)
}

// DueJobs returns active jobs whose scheduled time has arrived.
func (s *Scheduler) DueJobs(at time.Time) ([]*Job, error) {
	active, err := s.ListActiveJobs()
	if err != nil {
		return nil, err
	}
	var due []*Job
	for _, j := range active {
		if !j.ScheduledAt.After(at) {
			due = append(due, j)
		}
	}
	return due, nil
}

func scanJob(row *sql.Row) (*Job, error) {
	var j Job
	var scheduledAt, createdAt, updatedAt int64
	if err := row.Scan(&j.ID, &j.Target, &j.Version, &scheduledAt, &j.Status, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	j.ScheduledAt = time.Unix(scheduledAt, 0)
	j.CreatedAt = time.Unix(createdAt, 0)
	j.UpdatedAt = time.Unix(updatedAt, 0)
	return &j, nil
}

func scanJobRows(rows *sql.Rows) (*Job, error) {
	var j Job
	var scheduledAt, createdAt, updatedAt int64
	if err := rows.Scan(&j.ID, &j.Target, &j.Version, &scheduledAt, &j.Status, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	j.ScheduledAt = time.Unix(scheduledAt, 0)
	j.CreatedAt = time.Unix(createdAt, 0)
	j.UpdatedAt = time.Unix(updatedAt, 0)
	return &j, nil
}

// LockWindowStatus describes whether a point in time falls inside a lock
// window (including its configured buffers) and, if locked, when it frees up.
type LockWindowStatus struct {
	Locked bool
	FreeAt time.Time
}

const clockLayout = "3:04 PM"

// EvaluateLockWindow checks a single lock window against time t.
func EvaluateLockWindow(lw sourceloader.LockWindow, t time.Time) (LockWindowStatus, error) {
	if lw.ConditionScript != "" {
		return evaluateConditionWindow(lw, t)
	}
	return evaluateFixedWindow(lw, t)
}

func evaluateFixedWindow(lw sourceloader.LockWindow, t time.Time) (LockWindowStatus, error) {
	if lw.Day != "" && lw.Day != "daily" && lw.Day != "*" && !strings.EqualFold(lw.Day, t.Weekday().String()) {
		return LockWindowStatus{Locked: false}, nil
	}

	start, err := parseClock(lw.Start, t)
	if err != nil {
		return LockWindowStatus{}, fmt.Errorf("lock window %q: invalid start: %w", lw.Name, err)
	}
	end, err := parseClock(lw.End, t)
	if err != nil {
		return LockWindowStatus{}, fmt.Errorf("lock window %q: invalid end: %w", lw.Name, err)
	}
	if end.Before(start) {
		end = end.Add(24 * time.Hour)
	}

	blockedStart := start.Add(-time.Duration(lw.BufferBeforeMinutes) * time.Minute)
	blockedEnd := end.Add(time.Duration(lw.BufferAfterMinutes) * time.Minute)

	if t.Before(blockedStart) || !t.Before(blockedEnd) {
		return LockWindowStatus{Locked: false}, nil
	}
	return LockWindowStatus{Locked: true, FreeAt: blockedEnd}, nil
}

func evaluateConditionWindow(lw sourceloader.LockWindow, t time.Time) (LockWindowStatus, error) {
	cmd := exec.Command(lw.ConditionScript)
	err := cmd.Run()
	if err == nil {
		return LockWindowStatus{Locked: false}, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return LockWindowStatus{
			Locked: true,
			FreeAt: t.Add(time.Duration(lw.BufferAfterMinutes) * time.Minute),
		}, nil
	}
	return LockWindowStatus{}, fmt.Errorf("lock window %q: condition script failed to run: %w", lw.Name, err)
}

func parseClock(value string, ref time.Time) (time.Time, error) {
	parsed, err := time.Parse(clockLayout, value)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(ref.Year(), ref.Month(), ref.Day(), parsed.Hour(), parsed.Minute(), 0, 0, ref.Location()), nil
}

// EvaluateAllWindows checks every configured lock window and returns the most
// restrictive result: locked if ANY window is locked, FreeAt is the latest
// free time among all currently blocking windows.
func EvaluateAllWindows(windows []sourceloader.LockWindow, t time.Time) (LockWindowStatus, error) {
	result := LockWindowStatus{Locked: false}
	for _, lw := range windows {
		status, err := EvaluateLockWindow(lw, t)
		if err != nil {
			return LockWindowStatus{}, err
		}
		if status.Locked {
			result.Locked = true
			if status.FreeAt.After(result.FreeAt) {
				result.FreeAt = status.FreeAt
			}
		}
	}
	return result, nil
}

// IsLocked evaluates all lock windows configured in the loaded Source
// against time t.
func (s *Scheduler) IsLocked(t time.Time) (LockWindowStatus, error) {
	return EvaluateAllWindows(s.source.Config.LockWindows, t)
}
