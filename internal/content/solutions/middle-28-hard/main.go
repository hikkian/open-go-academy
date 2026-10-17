package main

import (
	"context"
)

type TaskService struct {
	repo TaskRepository
}

func NewTaskService(repo TaskRepository) (*TaskService, error) {
	if repo == nil {
		return nil, ErrRepositoryMissing
	}
	return &TaskService{repo: repo}, nil
}

func (s *TaskService) Get(ctx context.Context, id string) (Task, error) {
	return s.repo.Get(ctx, id)
}
