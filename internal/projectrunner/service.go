package projectrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"

	"academy/internal/runner"
)

type Service struct{ engine Engine }

func NewService(e Engine) *Service { return &Service{engine: e} }
func (s *Service) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		if !s.engine.Available(r.Context()) {
			http.Error(w, "unavailable", 503)
			return
		}
		w.WriteHeader(200)
		_, _ = io.WriteString(w, "ok")
	})
	m.HandleFunc("GET /capabilities", func(w http.ResponseWriter, r *http.Request) {
		// Capabilities are immutable for this worker process. Health is checked
		// separately by the API before it consumes this metadata.
		projects := []string{"p2"}
		profiles := map[string]EvaluationProfile{}
		if capability, ok := s.engine.(interface{ SupportsProject(string) bool }); ok && capability.SupportsProject("p3") {
			if provider, ok := s.engine.(interface {
				ProjectProfile(string) (EvaluationProfile, bool)
			}); ok {
				if profile, profileOK := provider.ProjectProfile("p3"); profileOK {
					projects = append(projects, "p3")
					profiles["p3"] = profile
				}
			}
		}
		if capability, ok := s.engine.(interface{ SupportsProject(string) bool }); ok && capability.SupportsProject("p4") {
			if provider, ok := s.engine.(interface {
				ProjectProfile(string) (EvaluationProfile, bool)
			}); ok {
				if profile, profileOK := provider.ProjectProfile("p4"); profileOK {
					projects = append(projects, "p4")
					profiles["p4"] = profile
				}
			}
		}
		if capability, ok := s.engine.(interface{ SupportsProject(string) bool }); ok && capability.SupportsProject("p5") {
			if provider, ok := s.engine.(interface {
				ProjectProfile(string) (EvaluationProfile, bool)
			}); ok {
				if profile, profileOK := provider.ProjectProfile("p5"); profileOK {
					projects = append(projects, "p5")
					profiles["p5"] = profile
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			Projects []string                     `json:"projects"`
			Profiles map[string]EvaluationProfile `json:"profiles"`
		}{Projects: projects, Profiles: profiles})
	})
	m.HandleFunc("POST /execute", func(w http.ResponseWriter, r *http.Request) {
		if !s.engine.Available(r.Context()) {
			http.Error(w, "unavailable", 503)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, MaxBody)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		var q Request
		if err := dec.Decode(&q); err != nil {
			http.Error(w, "invalid request", 400)
			return
		}
		var extra any
		if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
			http.Error(w, "invalid request", 400)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 180*time.Second)
		defer cancel()
		result, err := s.engine.Execute(ctx, q)
		if err != nil {
			if err.Error() == "project worker busy" {
				http.Error(w, "busy", 503)
			} else {
				http.Error(w, "invalid or unavailable job", 422)
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		response := struct {
			runner.Result
			Profile *EvaluationProfile `json:"evaluation_profile,omitempty"`
		}{Result: result}
		if provider, ok := s.engine.(interface {
			ProjectProfile(string) (EvaluationProfile, bool)
		}); ok {
			if profile, profileOK := provider.ProjectProfile(q.ProjectID); profileOK {
				response.Profile = &profile
			}
		}
		_ = json.NewEncoder(w).Encode(response)
	})
	return m
}

func Listen(socket string) (net.Listener, error) {
	if socket == "" {
		return nil, errors.New("socket path required")
	}
	lock, err := os.OpenFile(socket+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = lock.Chmod(0600); err != nil {
		lock.Close()
		return nil, err
	}
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		lock.Close()
		return nil, errors.New("project runner already running")
	}
	if fi, err := os.Lstat(socket); err == nil {
		if fi.Mode()&os.ModeSocket == 0 {
			_ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
			_ = lock.Close()
			return nil, errors.New("socket path exists and is not a socket")
		}
		c, e := net.DialTimeout("unix", socket, time.Second)
		if e == nil {
			c.Close()
			_ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
			_ = lock.Close()
			return nil, errors.New("project runner already running")
		}
		if e = os.Remove(socket); e != nil {
			_ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
			_ = lock.Close()
			return nil, e
		}
	} else if !os.IsNotExist(err) {
		_ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
		_ = lock.Close()
		return nil, err
	}
	l, err := net.Listen("unix", socket)
	if err != nil {
		_ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
		_ = lock.Close()
		return nil, err
	}
	if err = os.Chmod(socket, 0600); err != nil {
		l.Close()
		os.Remove(socket)
		_ = syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
		_ = lock.Close()
		return nil, err
	}
	return &lockedListener{Listener: l, lock: lock}, nil
}

type lockedListener struct {
	net.Listener
	lock *os.File
}

func (l *lockedListener) Close() error {
	err := l.Listener.Close()
	_ = syscall.Flock(int(l.lock.Fd()), syscall.LOCK_UN)
	if x := l.lock.Close(); err == nil {
		err = x
	}
	return err
}

type Client struct{ http *http.Client }

func NewClient(socket string) *Client {
	return &Client{http: &http.Client{Timeout: 220 * time.Second, Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}}}
}
func (c *Client) Available(ctx context.Context) bool {
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://project-runner/health", nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}
func (c *Client) SupportsProject(ctx context.Context, projectID string) bool {
	if projectID != "p2" && projectID != "p3" && projectID != "p4" && projectID != "p5" {
		return false
	}
	capabilities, ok := c.projectCapabilities(ctx)
	if !ok {
		return false
	}
	for _, supported := range capabilities.Projects {
		if supported == projectID {
			return true
		}
	}
	return false
}

func (c *Client) ProjectProfile(ctx context.Context, projectID string) (EvaluationProfile, bool) {
	if projectID != "p3" && projectID != "p4" && projectID != "p5" {
		return EvaluationProfile{}, false
	}
	capabilities, ok := c.projectCapabilities(ctx)
	if !ok {
		return EvaluationProfile{}, false
	}
	profile, ok := capabilities.Profiles[projectID]
	return profile, ok
}

type capabilityResponse struct {
	Projects []string                     `json:"projects"`
	Profiles map[string]EvaluationProfile `json:"profiles"`
}

func (c *Client) projectCapabilities(ctx context.Context) (capabilityResponse, bool) {
	req, err := http.NewRequestWithContext(ctx, "GET", "http://project-runner/capabilities", nil)
	if err != nil {
		return capabilityResponse{}, false
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return capabilityResponse{}, false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return capabilityResponse{}, false
	}
	const maxResponse = 4096
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse+1))
	if err != nil || len(body) > maxResponse {
		return capabilityResponse{}, false
	}
	var capabilities capabilityResponse
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&capabilities); err != nil {
		return capabilityResponse{}, false
	}
	var extra any
	if err = decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return capabilityResponse{}, false
	}
	if capabilities.Projects == nil || capabilities.Profiles == nil {
		return capabilityResponse{}, false
	}
	return capabilities, true
}
func (c *Client) Execute(ctx context.Context, q Request) (runner.Result, error) {
	result, _, err := c.executeWithProfile(ctx, q)
	return result, err
}

func (c *Client) ExecuteWithProfile(ctx context.Context, q Request) (runner.Result, EvaluationProfile, error) {
	result, profile, err := c.executeWithProfile(ctx, q)
	if err != nil {
		return runner.Result{}, EvaluationProfile{}, err
	}
	if profile == nil {
		return runner.Result{}, EvaluationProfile{}, errors.New("project runner omitted trusted evaluation profile")
	}
	return result, *profile, nil
}

func (c *Client) executeWithProfile(ctx context.Context, q Request) (runner.Result, *EvaluationProfile, error) {
	b, err := json.Marshal(q)
	if err != nil {
		return runner.Result{}, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", "http://project-runner/execute", bytes.NewReader(b))
	if err != nil {
		return runner.Result{}, nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return runner.Result{}, nil, errors.New("project runner unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return runner.Result{}, nil, errors.New("project runner rejected request")
	}
	const maxResponse = 1 << 20
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse+1))
	if err != nil || len(body) > maxResponse {
		return runner.Result{}, nil, errors.New("invalid project runner response")
	}
	var out struct {
		runner.Result
		Profile *EvaluationProfile `json:"evaluation_profile,omitempty"`
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if err = d.Decode(&out); err != nil {
		return runner.Result{}, nil, err
	}
	var extra any
	if err = d.Decode(&extra); !errors.Is(err, io.EOF) {
		return runner.Result{}, nil, errors.New("invalid project runner response")
	}
	return out.Result, out.Profile, nil
}
