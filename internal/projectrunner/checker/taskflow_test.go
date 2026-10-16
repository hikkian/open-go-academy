package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTaskFlowCheckerExercisesTrustedHTTPFlowAndRestartState(t *testing.T) {
	const (
		userA      = "123e4567-e89b-42d3-a456-426614174001"
		userB      = "123e4567-e89b-42d3-a456-426614174002"
		workspaceA = "123e4567-e89b-42d3-a456-426614174003"
		workspaceB = "123e4567-e89b-42d3-a456-426614174004"
		projectA   = "123e4567-e89b-42d3-a456-426614174005"
		projectB   = "123e4567-e89b-42d3-a456-426614174006"
		taskA      = "123e4567-e89b-42d3-a456-426614174007"
		taskB      = "123e4567-e89b-42d3-a456-426614174008"
		commentA   = "123e4567-e89b-42d3-a456-426614174009"
		commentB   = "123e4567-e89b-42d3-a456-426614174010"
	)
	type account struct{ id, password, workspace string }
	accounts := map[string]account{}
	sessions := map[string]string{}
	roles := map[string]map[string]string{}
	projects := map[string]string{}
	tasks := map[string]taskFlowTask{}
	comments := map[string][]taskFlowComment{}
	keyTasks := map[string]string{}
	nextProject, nextTask, nextComment := 0, 0, 0
	respond := func(w http.ResponseWriter, status int, value any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if value != nil {
			_ = json.NewEncoder(w).Encode(value)
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/readyz" || r.URL.Path == "/livez" {
			respond(w, 200, map[string]string{"status": "ready"})
			return
		}
		var body map[string]any
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		if r.Method == "POST" && r.URL.Path == "/api/v1/users" {
			email, _ := body["email"].(string)
			password, _ := body["password"].(string)
			if email == "" || password == "" {
				respond(w, 400, nil)
				return
			}
			id, ws := userA, workspaceA
			if len(accounts) > 0 {
				id, ws = userB, workspaceB
			}
			accounts[email] = account{id, password, ws}
			roles[ws] = map[string]string{email: "owner"}
			sessions["session-"+email] = email
			http.SetCookie(w, &http.Cookie{Name: "taskflow_session", Value: "session-" + email, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode})
			respond(w, 201, map[string]string{"status": "created"})
			return
		}
		if r.Method == "POST" && r.URL.Path == "/api/v1/sessions" {
			email, _ := body["email"].(string)
			password, _ := body["password"].(string)
			a, ok := accounts[email]
			if !ok || a.password != password {
				respond(w, 401, nil)
				return
			}
			sessions["session-"+email] = email
			http.SetCookie(w, &http.Cookie{Name: "taskflow_session", Value: "session-" + email, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode})
			respond(w, 200, map[string]string{"status": "ok"})
			return
		}
		if r.Method == "DELETE" && r.URL.Path == "/api/v1/sessions/current" {
			c, _ := r.Cookie("taskflow_session")
			if c != nil {
				delete(sessions, c.Value)
			}
			http.SetCookie(w, &http.Cookie{Name: "taskflow_session", Value: "", Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode, MaxAge: -1})
			w.WriteHeader(204)
			return
		}
		c, _ := r.Cookie("taskflow_session")
		email := ""
		if c != nil {
			email = sessions[c.Value]
		}
		actor, authenticated := accounts[email]
		if !authenticated {
			respond(w, 401, nil)
			return
		}
		if r.Method == "GET" && r.URL.Path == "/api/v1/workspaces" {
			respond(w, 200, taskFlowPage[taskFlowWorkspace]{Items: []taskFlowWorkspace{{actor.workspace, "workspace", "owner"}}})
			return
		}
		if r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/api/v1/workspaces/") && strings.HasSuffix(r.URL.Path, "/projects") {
			workspace := strings.Split(strings.Trim(r.URL.Path, "/"), "/")[3]
			if roles[workspace][email] != "owner" && roles[workspace][email] != "editor" {
				respond(w, 404, nil)
				return
			}
			nextProject++
			id := projectA
			if nextProject == 2 {
				id = projectB
			}
			projects[id] = workspace
			respond(w, 201, map[string]string{"id": id, "workspace_id": workspace, "name": "project"})
			return
		}
		if r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/api/v1/workspaces/") && strings.HasSuffix(r.URL.Path, "/projects") {
			parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
			workspace := parts[3]
			if roles[workspace][email] == "" {
				respond(w, 200, taskFlowPage[taskFlowProject]{Items: []taskFlowProject{}})
				return
			}
			items := []taskFlowProject{}
			for id, ws := range projects {
				if ws == workspace {
					items = append(items, taskFlowProject{id, ws, "project"})
				}
			}
			respond(w, 200, taskFlowPage[taskFlowProject]{Items: items})
			return
		}
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "/members/") {
			parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
			workspace := parts[3]
			target := parts[5]
			role, _ := body["role"].(string)
			if roles[workspace][email] != "owner" || accounts[target].id == "" || (role != "viewer" && role != "editor") {
				respond(w, 404, nil)
				return
			}
			roles[workspace][target] = role
			respond(w, 200, map[string]string{"status": "updated"})
			return
		}
		if r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/api/v1/projects/") && strings.HasSuffix(r.URL.Path, "/tasks") {
			if _, spoof := body["user_id"]; spoof {
				respond(w, 400, nil)
				return
			}
			parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
			project := parts[3]
			workspace := projects[project]
			role := roles[workspace][email]
			if workspace == "" || role != "owner" && role != "editor" {
				respond(w, 404, nil)
				return
			}
			key, _ := body["idempotency_key"].(string)
			scope := project + "/" + key
			if id := keyTasks[scope]; id != "" {
				respond(w, 200, map[string]string{"id": id})
				return
			}
			nextTask++
			id := taskA
			if nextTask > 1 {
				id = taskB
			}
			keyTasks[scope] = id
			title, _ := body["title"].(string)
			tasks[id] = taskFlowTask{ID: id, ProjectID: project, Title: title, Description: "", State: "open", Version: 1}
			respond(w, 201, map[string]string{"id": id})
			return
		}
		if r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/api/v1/projects/") && strings.HasSuffix(r.URL.Path, "/tasks") {
			parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
			project := parts[3]
			if roles[projects[project]][email] == "" {
				respond(w, 200, taskFlowPage[taskFlowTask]{Items: []taskFlowTask{}})
				return
			}
			items := []taskFlowTask{}
			for _, task := range tasks {
				if task.ProjectID == project {
					items = append(items, task)
				}
			}
			respond(w, 200, taskFlowPage[taskFlowTask]{Items: items})
			return
		}
		if r.Method == "PATCH" && strings.HasPrefix(r.URL.Path, "/api/v1/tasks/") {
			id := strings.TrimPrefix(r.URL.Path, "/api/v1/tasks/")
			task := tasks[id]
			role := roles[projects[task.ProjectID]][email]
			version, _ := body["version"].(float64)
			state, _ := body["state"].(string)
			if task.ID == "" || role != "owner" && role != "editor" || int(version) != task.Version {
				respond(w, 404, nil)
				return
			}
			task.State = state
			task.Version++
			tasks[id] = task
			respond(w, 200, task)
			return
		}
		if r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/api/v1/tasks/") && strings.HasSuffix(r.URL.Path, "/comments") {
			id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/tasks/"), "/comments")
			task := tasks[id]
			role := roles[projects[task.ProjectID]][email]
			if task.ID == "" || role != "owner" && role != "editor" {
				respond(w, 404, nil)
				return
			}
			nextComment++
			commentID := commentA
			if nextComment > 1 {
				commentID = commentB
			}
			text, _ := body["body"].(string)
			comments[id] = append(comments[id], taskFlowComment{ID: commentID, AuthorID: actor.id, Body: text, CreatedAt: "2026-01-01T00:00:00Z"})
			respond(w, 201, map[string]string{"id": commentID})
			return
		}
		if r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/api/v1/tasks/") && strings.HasSuffix(r.URL.Path, "/comments") {
			id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/tasks/"), "/comments")
			if tasks[id].ID == "" || roles[projects[tasks[id].ProjectID]][email] == "" {
				respond(w, 200, taskFlowPage[taskFlowComment]{Items: []taskFlowComment{}})
				return
			}
			respond(w, 200, taskFlowPage[taskFlowComment]{Items: comments[id]})
			return
		}
		respond(w, 404, nil)
	}))
	defer server.Close()
	c, err := newChecker(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	state := c.taskFlowInitial()
	if len(c.checks) == 0 {
		t.Fatal("checker produced no checks")
	}
	for _, item := range c.checks {
		if !item.Passed {
			t.Errorf("initial check failed: %s", item.Name)
		}
	}
	b, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := newChecker(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	persisted.taskFlowPersistence(b)
	for _, item := range persisted.checks {
		if !item.Passed {
			t.Errorf("persistence check failed: %s", item.Name)
		}
	}
}

func TestTaskFlowCheckerRejectsExternalAndUntrustedProtocolTargets(t *testing.T) {
	for _, target := range []string{"https://127.0.0.1:8098", "http://example.test:8098", "http://127.0.0.1:8098/path", "http://user@127.0.0.1:8098"} {
		if _, err := newChecker(target); err == nil {
			t.Errorf("trusted checker accepted non-loopback/invalid target %q", target)
		}
	}
	if _, err := parseRequest([]byte(`{"project_id":"p3","phase":"health","base_url":"http://127.0.0.1:8098","extra":true}`)); err == nil {
		t.Fatal("checker protocol accepted an unknown field")
	}
}
