// Package scheduler provides cron-based scheduling for speedtests.
package scheduler

import (
	"context"
	"fmt"
	"sync"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/lan-dot-party/flowgauge/internal/config"
	"github.com/lan-dot-party/flowgauge/internal/speedtest"
	"github.com/lan-dot-party/flowgauge/internal/storage"
)

// Scheduler manages scheduled speedtest jobs.
type Scheduler struct {
	cron     *cron.Cron
	config   *config.SchedulerConfig
	runner   *speedtest.MultiWANRunner
	storage  storage.Storage
	logger   *zap.Logger
	running  bool
	mu       sync.Mutex
	jobID    cron.EntryID
}

// NewScheduler creates a new scheduler instance.
func NewScheduler(cfg *config.SchedulerConfig, runner *speedtest.MultiWANRunner, store storage.Storage, logger *zap.Logger) (*Scheduler, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	if cfg == nil {
		return nil, fmt.Errorf("scheduler config is required")
	}

	if runner == nil {
		return nil, fmt.Errorf("speedtest runner is required")
	}

	if store == nil {
		return nil, fmt.Errorf("storage is required")
	}

	// Create cron with seconds support (optional) and logger
	c := cron.New(
		cron.WithLogger(cron.VerbosePrintfLogger(&cronLogger{logger: logger})),
		cron.WithChain(
			cron.Recover(cron.VerbosePrintfLogger(&cronLogger{logger: logger})),
		),
	)

	return &Scheduler{
		cron:    c,
		config:  cfg,
		runner:  runner,
		storage: store,
		logger:  logger,
	}, nil
}

// Start begins the scheduler.
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler already running")
	}

	if !s.config.Enabled {
		s.logger.Info("Scheduler is disabled in configuration")
		return nil
	}

	// Create the speedtest job
	job := NewSpeedtestJob(s.runner, s.storage, s.logger)

	// Add the job to cron
	entryID, err := s.cron.AddFunc(s.config.Schedule, job.Run)
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w (schedule: %s)", err, s.config.Schedule)
	}
	s.jobID = entryID

	// Start the cron scheduler
	s.cron.Start()
	s.running = true

	s.logger.Info("Scheduler started",
		zap.String("schedule", s.config.Schedule),
		zap.Int("entry_id", int(entryID)),
	)

	// Log next run time
	entry := s.cron.Entry(entryID)
	s.logger.Info("Next scheduled run",
		zap.Time("next_run", entry.Next),
	)

	return nil
}

// Stop gracefully stops the scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	ctx := s.cron.Stop()
	<-ctx.Done()
	s.running = false

	s.logger.Info("Scheduler stopped")
}

// IsRunning returns whether the scheduler is currently running.
func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

// NextRun returns the next scheduled run time.
func (s *Scheduler) NextRun() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running || s.jobID == 0 {
		return "not scheduled"
	}

	entry := s.cron.Entry(s.jobID)
	return entry.Next.Format("2006-01-02 15:04:05")
}

// TriggerNow manually triggers a speedtest run.
func (s *Scheduler) TriggerNow() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.jobID == 0 {
		s.logger.Warn("Cannot trigger: no job configured")
		return
	}

	entry := s.cron.Entry(s.jobID)
	if entry.Job != nil {
		s.logger.Info("Manually triggering speedtest")
		go entry.Job.Run()
	}
}

// GetStatus returns the current scheduler status.
func (s *Scheduler) GetStatus() Status {
	s.mu.Lock()
	defer s.mu.Unlock()

	status := Status{
		Enabled:  s.config.Enabled,
		Running:  s.running,
		Schedule: s.config.Schedule,
	}

	if s.running && s.jobID != 0 {
		entry := s.cron.Entry(s.jobID)
		status.NextRun = entry.Next.Format("2006-01-02 15:04:05")
		if !entry.Prev.IsZero() {
			status.LastRun = entry.Prev.Format("2006-01-02 15:04:05")
		}
	}

	return status
}

// Status represents the scheduler status.
type Status struct {
	Enabled  bool   `json:"enabled"`
	Running  bool   `json:"running"`
	Schedule string `json:"schedule"`
	NextRun  string `json:"next_run,omitempty"`
	LastRun  string `json:"last_run,omitempty"`
}

// cronLogger adapts zap.Logger to cron's logger interface.
type cronLogger struct {
	logger *zap.Logger
}

func (l *cronLogger) Printf(format string, args ...interface{}) {
	l.logger.Debug(fmt.Sprintf(format, args...))
}

// RunOnce runs the speedtest job once immediately (useful for testing).
func (s *Scheduler) RunOnce(ctx context.Context) error {
	job := NewSpeedtestJob(s.runner, s.storage, s.logger)
	return job.RunWithContext(ctx)
}

