package main

import (
	"context"
)

type Task struct {
	ID         string `json:"id"`
	Repository string `json:"repository"`
}

type TaskRepository interface {
	Get(context.Context, string) (Task, error)
}

type RepositoryMissingError struct{}

func (RepositoryMissingError) Error() string { return "repository missing" }

type TaskService struct {
	repo TaskRepository
}

func NewTaskService(repo TaskRepository) (*TaskService, error) {
	if repo == nil {
		return nil, RepositoryMissingError{}
	}
	return &TaskService{repo: repo}, nil
}

func (s *TaskService) Get(ctx context.Context, id string) (Task, error) {
	return s.repo.Get(ctx, id)
}
