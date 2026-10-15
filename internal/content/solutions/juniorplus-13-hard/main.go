package main

import (
	"context"
	"errors"
)

func RunRepository(ctx context.Context, load func(context.Context) error) string {
	err := load(ctx)
	if err == nil {
		return "ok"
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "cancelled"
	}
	return "repository_error"
}
