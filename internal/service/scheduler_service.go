package service

import (
	"context"

	"go.uber.org/zap"

	"lark-robot/internal/model"
	"lark-robot/internal/repository"
	"lark-robot/internal/scheduler"
)

type SchedulerService struct {
	repo      *repository.ScheduledTaskRepo
	scheduler *scheduler.Scheduler
	logger    *zap.Logger
}

func NewSchedulerService(repo *repository.ScheduledTaskRepo, sched *scheduler.Scheduler, logger *zap.Logger) *SchedulerService {
	return &SchedulerService{
		repo:      repo,
		scheduler: sched,
		logger:    logger,
	}
}

// LoadAndStartAll loads all enabled tasks from DB and registers them with the scheduler.
func (s *SchedulerService) LoadAndStartAll() error {
	tasks, err := s.repo.ListEnabled()
	if err != nil {
		return err
	}
	for _, task := range tasks {
		t := task
		if err := s.scheduler.AddTask(&t); err != nil {
			s.logger.Error("failed to add scheduled task", zap.Uint("id", task.ID), zap.Error(err))
		}
	}
	s.logger.Info("loaded scheduled tasks", zap.Int("count", len(tasks)))
	return nil
}

func (s *SchedulerService) List(page, pageSize int) ([]model.ScheduledTask, int64, error) {
	return s.repo.List(page, pageSize)
}

func (s *SchedulerService) GetByID(id uint) (*model.ScheduledTask, error) {
	return s.repo.GetByID(id)
}

func (s *SchedulerService) Create(task *model.ScheduledTask) error {
	if err := s.repo.Create(task); err != nil {
		return err
	}
	if task.Enabled {
		return s.scheduler.AddTask(task)
	}
	return nil
}

func (s *SchedulerService) Update(task *model.ScheduledTask) error {
	if err := s.repo.Update(task); err != nil {
		return err
	}
	return s.scheduler.ReloadTask(task)
}

func (s *SchedulerService) Delete(id uint) error {
	s.scheduler.RemoveTask(id)
	return s.repo.Delete(id)
}

func (s *SchedulerService) Toggle(id uint) error {
	if err := s.repo.ToggleEnabled(id); err != nil {
		return err
	}
	task, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	return s.scheduler.ReloadTask(task)
}

// RunNow triggers a task immediately (for testing).
func (s *SchedulerService) RunNow(ctx context.Context, id uint) error {
	task, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	return s.scheduler.RunTaskNow(ctx, task)
}

// TaskCount returns total number of tasks.
func (s *SchedulerService) TaskCount() (int64, error) {
	_, total, err := s.repo.List(1, 1)
	return total, err
}
