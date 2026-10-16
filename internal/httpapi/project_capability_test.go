package httpapi

import (
	"context"
	"testing"

	"academy/internal/projectrunner"
	"academy/internal/runner"
)

type capabilityExecutor struct {
	available bool
	p3        bool
}

func (e capabilityExecutor) Available(context.Context) bool { return e.available }
func (e capabilityExecutor) Execute(context.Context, projectrunner.Request) (runner.Result, error) {
	return runner.Result{}, nil
}
func (e capabilityExecutor) SupportsProject(_ context.Context, id string) bool {
	return id == "p3" && e.p3
}

func TestTaskFlowAvailabilityRequiresHealthyAdvertisedCapability(t *testing.T) {
	tests := []struct {
		name string
		exec ProjectExecutor
		want bool
	}{
		{name: "nil worker", exec: nil, want: false},
		{name: "unhealthy worker", exec: capabilityExecutor{available: false, p3: true}, want: false},
		{name: "P2-only profile", exec: capabilityExecutor{available: true}, want: false},
		{name: "combined P2/P3 profile", exec: capabilityExecutor{available: true, p3: true}, want: true},
		{name: "legacy fake executor cannot advertise P3", exec: &p2Executor{}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &API{ProjectRunner: tt.exec}
			if got := a.taskFlowAvailable(context.Background()); got != tt.want {
				t.Fatalf("taskFlowAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}
