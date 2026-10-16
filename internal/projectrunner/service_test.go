package projectrunner

import (
	"context"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"academy/internal/runner"
)

type capabilityTestEngine struct{ p3 bool }

func (e capabilityTestEngine) Available(context.Context) bool { return true }
func (e capabilityTestEngine) Execute(context.Context, Request) (runner.Result, error) {
	return runner.Result{Status: "ok"}, nil
}
func (e capabilityTestEngine) SupportsProject(projectID string) bool {
	return projectID == "p2" || projectID == "p3" && e.p3
}
func (e capabilityTestEngine) ProjectProfile(projectID string) (EvaluationProfile, bool) {
	if projectID != "p3" || !e.p3 {
		return EvaluationProfile{}, false
	}
	return EvaluationProfile{ProjectVersion: "p3-test-v1", CriteriaHash: "sha256:test", CheckerVersion: "taskflow-test-v1", CheckerDigest: "sha256:test", RuntimePolicyHash: "sha256:test", RubricVersion: "rubric-test-v1", RubricHash: "sha256:test"}, true
}

func TestClientReadsFailClosedProjectCapabilities(t *testing.T) {
	for _, tc := range []struct {
		name string
		p3   bool
		want bool
	}{
		{name: "p2-only profile", p3: false, want: false},
		{name: "combined profile", p3: true, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			socket := filepath.Join(t.TempDir(), "project-runner.sock")
			listener, err := Listen(socket)
			if err != nil {
				t.Fatal(err)
			}
			server := &http.Server{Handler: NewService(capabilityTestEngine{p3: tc.p3}).Handler()}
			serveErr := make(chan error, 1)
			go func() { serveErr <- server.Serve(listener) }()
			t.Cleanup(func() {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				_ = server.Shutdown(ctx)
				if err := <-serveErr; err != nil && err != http.ErrServerClosed {
					t.Errorf("serve: %v", err)
				}
			})

			client := NewClient(socket)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if !client.Available(ctx) {
				t.Fatal("project runner health check failed")
			}
			if !client.SupportsProject(ctx, "p2") {
				t.Fatal("P2 capability missing")
			}
			if got := client.SupportsProject(ctx, "p3"); got != tc.want {
				t.Fatalf("P3 capability = %v, want %v", got, tc.want)
			}
			profile, profileOK := client.ProjectProfile(ctx, "p3")
			if profileOK != tc.want {
				t.Fatalf("P3 evaluation profile availability = %v, want %v", profileOK, tc.want)
			}
			if tc.want {
				if profile.ProjectVersion == "" || profile.CheckerDigest == "" || profile.RuntimePolicyHash == "" || profile.RubricHash == "" {
					t.Fatalf("trusted profile is incomplete: %+v", profile)
				}
				result, executedProfile, err := client.ExecuteWithProfile(ctx, Request{ProjectID: "p3", Action: "submit", Files: map[string]string{"main.go": "package main\n"}})
				if err != nil || result.Status != "ok" || executedProfile != profile {
					t.Fatalf("execution did not bind its trusted profile: result=%+v profile=%+v err=%v", result, executedProfile, err)
				}
			}
			if client.SupportsProject(ctx, "unknown") {
				t.Fatal("unknown project capability accepted")
			}
		})
	}
}
