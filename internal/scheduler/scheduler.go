package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
)

// RedisClient defines the Redis operations needed by the scheduler.
type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

// Callbacks defines the functions the scheduler will invoke.
type Callbacks interface {
	Remind(ctx context.Context) error
	Chase(ctx context.Context) error
	Summary(ctx context.Context) error
}

// Scheduler manages scheduled jobs for the management loop.
type Scheduler struct {
	scheduler gocron.Scheduler
	redis     RedisClient
	callbacks Callbacks
	jobs      []jobInfo
}

type jobInfo struct {
	name     string
	cron     string
	callback func(ctx context.Context) error
	jobID    uuid.UUID
}

// New creates a new scheduler with remind/chase/summary jobs.
func New(timezone string, redis RedisClient, callbacks Callbacks) (*Scheduler, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("load timezone %q: %w", timezone, err)
	}

	s, err := gocron.NewScheduler(gocron.WithLocation(loc))
	if err != nil {
		return nil, fmt.Errorf("create scheduler: %w", err)
	}

	sched := &Scheduler{
		scheduler: s,
		redis:     redis,
		callbacks: callbacks,
	}

	// Register jobs
	jobs := []struct {
		name string
		cron string
		fn   func(ctx context.Context) error
	}{
		{"remind", "0 9 * * *", callbacks.Remind},    // 9:00 AM
		{"chase", "30 17 * * *", callbacks.Chase},    // 5:30 PM
		{"summary", "0 19 * * *", callbacks.Summary}, // 7:00 PM
	}

	for _, j := range jobs {
		jName := j.name
		jFn := j.fn
		gj, err := s.NewJob(
			gocron.CronJob(j.cron, false),
			gocron.NewTask(func() {
				ctx := context.Background()
				slog.Info("scheduler job running", "job", jName)
				if err := jFn(ctx); err != nil {
					slog.Error("scheduler job failed", "job", jName, "error", err)
				}
				// Update last_run
				redis.Set(ctx, fmt.Sprintf("scheduler:last_run:%s", jName),
					time.Now().Format(time.RFC3339), 0)
			}),
		)
		if err != nil {
			return nil, fmt.Errorf("register job %s: %w", jName, err)
		}
		sched.jobs = append(sched.jobs, jobInfo{name: jName, cron: j.cron, callback: jFn, jobID: gj.ID()})
	}

	return sched, nil
}

// Start begins the scheduler and checks for missed jobs.
func (s *Scheduler) Start(ctx context.Context) {
	slog.Info("scheduler starting")
	s.CheckMissedJobs(ctx)
	s.scheduler.Start()
}

// Stop gracefully shuts down the scheduler.
func (s *Scheduler) Stop() error {
	slog.Info("scheduler stopping")
	return s.scheduler.Shutdown()
}

// AddJob registers an additional cron job with the scheduler.
func (s *Scheduler) AddJob(name, cron string, fn func(ctx context.Context) error) error {
	jName := name
	jFn := fn
	redis := s.redis
	gj, err := s.scheduler.NewJob(
		gocron.CronJob(cron, false),
		gocron.NewTask(func() {
			ctx := context.Background()
			slog.Info("scheduler job running", "job", jName)
			if err := jFn(ctx); err != nil {
				slog.Error("scheduler job failed", "job", jName, "error", err)
			}
			redis.Set(ctx, fmt.Sprintf("scheduler:last_run:%s", jName),
				time.Now().Format(time.RFC3339), 0)
		}),
	)
	if err != nil {
		return fmt.Errorf("register job %s: %w", name, err)
	}
	s.jobs = append(s.jobs, jobInfo{name: name, cron: cron, callback: fn, jobID: gj.ID()})
	return nil
}

// JobCount returns the number of registered jobs.
func (s *Scheduler) JobCount() int {
	return len(s.jobs)
}

// JobInfo holds information about a scheduled job.
type JobInfo struct {
	Name    string    `json:"name"`
	Cron    string    `json:"cron"`
	LastRun time.Time `json:"last_run"`
	NextRun time.Time `json:"next_run"`
}

// ListJobs returns info about all registered jobs.
func (s *Scheduler) ListJobs() []JobInfo {
	result := make([]JobInfo, len(s.jobs))
	for i, j := range s.jobs {
		result[i] = JobInfo{
			Name:    j.name,
			Cron:    j.cron,
			LastRun: s.getLastRun(j.name),
			NextRun: s.getNextRun(j.jobID),
		}
	}
	return result
}

// UpdateJobSchedule updates the cron expression for an existing job.
func (s *Scheduler) UpdateJobSchedule(name, cron string) error {
	var found *jobInfo
	for i := range s.jobs {
		if s.jobs[i].name == name {
			found = &s.jobs[i]
			break
		}
	}
	if found == nil {
		return fmt.Errorf("job %q not found", name)
	}

	// Remove old job from gocron
	if err := s.scheduler.RemoveJob(found.jobID); err != nil {
		return fmt.Errorf("remove job %q: %w", name, err)
	}

	// Re-add with new schedule
	jName := found.name
	jFn := found.callback
	redis := s.redis
	gj, err := s.scheduler.NewJob(
		gocron.CronJob(cron, false),
		gocron.NewTask(func() {
			ctx := context.Background()
			slog.Info("scheduler job running", "job", jName)
			if err := jFn(ctx); err != nil {
				slog.Error("scheduler job failed", "job", jName, "error", err)
			}
			redis.Set(ctx, fmt.Sprintf("scheduler:last_run:%s", jName),
				time.Now().Format(time.RFC3339), 0)
		}),
	)
	if err != nil {
		return fmt.Errorf("re-add job %q with cron %q: %w", name, cron, err)
	}

	found.cron = cron
	found.jobID = gj.ID()
	return nil
}

// TriggerJob runs a job immediately by name.
func (s *Scheduler) TriggerJob(ctx context.Context, name string) error {
	for _, j := range s.jobs {
		if j.name == name {
			return j.callback(ctx)
		}
	}
	return fmt.Errorf("job %q not found", name)
}

// getLastRun returns the last run time from Redis.
func (s *Scheduler) getLastRun(name string) time.Time {
	raw, err := s.redis.Get(context.Background(), fmt.Sprintf("scheduler:last_run:%s", name))
	if err != nil {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}
	}
	return t
}

// getNextRun returns the next run time from gocron.
func (s *Scheduler) getNextRun(id uuid.UUID) time.Time {
	for _, gj := range s.scheduler.Jobs() {
		if gj.ID() == id {
			next, _ := gj.NextRun()
			return next
		}
	}
	return time.Time{}
}

// CheckMissedJobs checks if any jobs missed their window and runs catch-up.
func (s *Scheduler) CheckMissedJobs(ctx context.Context) {
	threshold := 2 * time.Hour

	for _, j := range s.jobs {
		key := fmt.Sprintf("scheduler:last_run:%s", j.name)
		raw, err := s.redis.Get(ctx, key)
		if err != nil {
			continue // no last_run = skip (first run or Redis error)
		}

		lastRun, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			slog.Warn("parse last_run", "job", j.name, "raw", raw, "error", err)
			continue
		}

		if time.Since(lastRun) > threshold {
			slog.Info("missed job detected, running catch-up", "job", j.name, "last_run", lastRun)
			if err := j.callback(ctx); err != nil {
				slog.Error("catch-up failed", "job", j.name, "error", err)
			}
			// Update last_run after catch-up
			s.redis.Set(ctx, key, time.Now().Format(time.RFC3339), 0)
		}
	}
}
