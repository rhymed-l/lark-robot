package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"lark-robot/internal/model"
)

// SendFunc is the function signature for sending a message.
type SendFunc func(ctx context.Context, chatID, msgType, content, source string) (string, error)

// UpdateLastRunFunc is called after a scheduled task runs.
type UpdateLastRunFunc func(id uint) error

type Scheduler struct {
	cron          *cron.Cron
	entries       map[uint]cron.EntryID
	mu            sync.Mutex
	sendFunc      SendFunc
	updateLastRun UpdateLastRunFunc
	logger        *zap.Logger
}

func New(sendFunc SendFunc, updateLastRun UpdateLastRunFunc, logger *zap.Logger) *Scheduler {
	c := cron.New(cron.WithSeconds())
	return &Scheduler{
		cron:          c,
		entries:       make(map[uint]cron.EntryID),
		sendFunc:      sendFunc,
		updateLastRun: updateLastRun,
		logger:        logger,
	}
}

// AddCleanupJob registers a cron job for maintenance tasks (e.g., log cleanup).
func (s *Scheduler) AddCleanupJob(cronExpr string, fn func()) error {
	_, err := s.cron.AddFunc(cronExpr, fn)
	return err
}

func (s *Scheduler) Start() {
	s.cron.Start()
	s.logger.Info("scheduler started")
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
	s.logger.Info("scheduler stopped")
}

func (s *Scheduler) AddTask(task *model.ScheduledTask) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	taskID := task.ID
	chatID := task.ChatID
	msgType := task.MsgType
	content := task.Content

	entryID, err := s.cron.AddFunc(task.CronExpr, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := s.sendFunc(ctx, chatID, msgType, content, "scheduled")
		if err != nil {
			s.logger.Error("scheduled task send failed",
				zap.Uint("task_id", taskID),
				zap.Error(err),
			)
			return
		}

		if err := s.updateLastRun(taskID); err != nil {
			s.logger.Error("failed to update last_run_at",
				zap.Uint("task_id", taskID),
				zap.Error(err),
			)
		}

		s.logger.Info("scheduled task executed", zap.Uint("task_id", taskID))
	})
	if err != nil {
		return fmt.Errorf("invalid cron expression %q: %w", task.CronExpr, err)
	}
	s.entries[task.ID] = entryID
	return nil
}

func (s *Scheduler) RemoveTask(taskID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entryID, ok := s.entries[taskID]; ok {
		s.cron.Remove(entryID)
		delete(s.entries, taskID)
	}
}

func (s *Scheduler) ReloadTask(task *model.ScheduledTask) error {
	s.RemoveTask(task.ID)
	if task.Enabled {
		return s.AddTask(task)
	}
	return nil
}

// RunTaskNow executes a task immediately, bypassing the schedule.
func (s *Scheduler) RunTaskNow(ctx context.Context, task *model.ScheduledTask) error {
	_, err := s.sendFunc(ctx, task.ChatID, task.MsgType, task.Content, "manual")
	if err != nil {
		return err
	}
	return s.updateLastRun(task.ID)
}
