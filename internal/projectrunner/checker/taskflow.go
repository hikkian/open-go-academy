package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
)

var taskFlowUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

type taskFlowState struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	WorkspaceID     string `json:"workspace_id"`
	ProjectID       string `json:"project_id"`
	SecondProjectID string `json:"second_project_id"`
	TaskID          string `json:"task_id"`
	CommentID       string `json:"comment_id"`
	IdempotencyKey  string `json:"idempotency_key"`
}
type taskFlowPage[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
}
type taskFlowWorkspace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}
type taskFlowProject struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
}
type taskFlowTask struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	Version     int    `json:"version"`
}
type taskFlowComment struct {
	ID        string `json:"id"`
	AuthorID  string `json:"author_id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

func (c *checker) taskFlowReadiness() {
	c.add("readyz", c.call("GET", "/readyz", nil, nil, 200, nil) == nil)
	c.taskFlowLiveness()
}

func (c *checker) taskFlowLiveness() {
	c.add("livez", c.call("GET", "/livez", nil, nil, 200, nil) == nil)
}

func (c *checker) taskFlowRegister(email, password string) bool {
	var out map[string]string
	err := c.call("POST", "/api/v1/users", map[string]string{"email": email, "password": password}, nil, 201, &out)
	cookie, ok := cookieByName(c.last.header, "taskflow_session")
	valid := err == nil && out["status"] == "created" && ok && cookie.Value != "" && cookie.HttpOnly && cookie.Secure && cookie.SameSite == http.SameSiteStrictMode && cookie.Path == "/"
	c.add("auth.register_server_owned_session", valid)
	return valid
}

func (c *checker) taskFlowInitial() any {
	s := taskFlowState{Email: "learner-" + randomHex(8) + "@example.test", Password: "Academy-" + randomHex(20) + "!", IdempotencyKey: "academy-p3-" + randomHex(12)}
	if !emailPattern.MatchString(s.Email) || len(s.Password) < 12 || s.IdempotencyKey == "academy-p3-" {
		c.add("fixture.generated", false)
		return nil
	}
	registered := c.taskFlowRegister(s.Email, s.Password)
	var workspaces taskFlowPage[taskFlowWorkspace]
	err := c.call("GET", "/api/v1/workspaces", nil, nil, 200, &workspaces)
	if err == nil && len(workspaces.Items) > 0 {
		s.WorkspaceID = workspaces.Items[0].ID
	}
	c.add("workspaces.list_owner", err == nil && taskFlowUUID.MatchString(s.WorkspaceID) && workspaces.Items[0].Role == "owner")
	var project taskFlowProject
	err = c.call("POST", "/api/v1/workspaces/"+url.PathEscape(s.WorkspaceID)+"/projects", map[string]string{"name": "Acceptance"}, nil, 201, &project)
	s.ProjectID = project.ID
	c.add("projects.create", err == nil && taskFlowUUID.MatchString(project.ID))
	var second taskFlowProject
	err = c.call("POST", "/api/v1/workspaces/"+url.PathEscape(s.WorkspaceID)+"/projects", map[string]string{"name": "Independent"}, nil, 201, &second)
	s.SecondProjectID = second.ID
	c.add("projects.create_independent", err == nil && taskFlowUUID.MatchString(second.ID) && second.ID != s.ProjectID)

	// A second user must not read or mutate the first user's project by UUID.
	aCookies := cloneCookies(c.cookies)
	bEmail := "other-" + randomHex(8) + "@example.test"
	bPassword := "Academy-" + randomHex(20) + "!"
	bRegistered := c.taskFlowRegister(bEmail, bPassword)
	_ = c.call("POST", "/api/v1/projects/"+url.PathEscape(s.ProjectID)+"/tasks", map[string]any{"title": "foreign", "description": "", "idempotency_key": "foreign-" + randomHex(6)}, nil, 404, nil)
	c.add("tenant.foreign_task_create_denied", c.last.status == 404 && bRegistered)
	var foreignProjects taskFlowPage[taskFlowProject]
	err = c.call("GET", "/api/v1/workspaces/"+url.PathEscape(s.WorkspaceID)+"/projects", nil, nil, 200, &foreignProjects)
	c.add("tenant.workspace_project_list_empty", err == nil && len(foreignProjects.Items) == 0)

	c.cookies = cloneCookies(aCookies)
	var created taskFlowTask
	err = c.call("POST", "/api/v1/projects/"+url.PathEscape(s.ProjectID)+"/tasks", map[string]any{"title": "Ship API slice", "description": "fixed trusted checker fixture", "idempotency_key": s.IdempotencyKey}, nil, 201, &created)
	s.TaskID = created.ID
	c.add("tasks.create", err == nil && taskFlowUUID.MatchString(s.TaskID))
	var replay taskFlowTask
	err = c.call("POST", "/api/v1/projects/"+url.PathEscape(s.ProjectID)+"/tasks", map[string]any{"title": "different retry body", "description": "ignored", "idempotency_key": s.IdempotencyKey}, nil, 200, &replay)
	c.add("tasks.idempotency_replay", err == nil && replay.ID == s.TaskID)
	var independent taskFlowTask
	err = c.call("POST", "/api/v1/projects/"+url.PathEscape(s.SecondProjectID)+"/tasks", map[string]any{"title": "same scoped key", "description": "", "idempotency_key": s.IdempotencyKey}, nil, 201, &independent)
	c.add("tasks.project_scoped_key", err == nil && taskFlowUUID.MatchString(independent.ID) && independent.ID != s.TaskID)
	// Client-supplied identity fields are rejected instead of becoming authority.
	err = c.call("POST", "/api/v1/projects/"+url.PathEscape(s.ProjectID)+"/tasks", map[string]any{"title": "spoof", "idempotency_key": "spoof-" + randomHex(6), "user_id": "00000000-0000-0000-0000-000000000000"}, nil, 400, nil)
	c.add("security.reject_client_identity", err == nil)

	var listed taskFlowPage[taskFlowTask]
	err = c.call("GET", "/api/v1/projects/"+url.PathEscape(s.ProjectID)+"/tasks", nil, nil, 200, &listed)
	found := false
	for _, task := range listed.Items {
		if task.ID == s.TaskID && task.Title == "Ship API slice" {
			found = true
		}
	}
	initialTask := false
	for _, task := range listed.Items {
		if task.ID == s.TaskID && task.State == "open" && task.Version == 1 {
			initialTask = true
		}
	}
	c.add("tasks.list_bounded", err == nil && found && initialTask && len(listed.Items) <= 100)
	var updated taskFlowTask
	err = c.call("PATCH", "/api/v1/tasks/"+url.PathEscape(s.TaskID), map[string]any{"state": "doing", "version": 1}, nil, 200, &updated)
	c.add("tasks.version_update", err == nil && updated.State == "doing" && updated.Version == 2)
	err = c.call("PATCH", "/api/v1/tasks/"+url.PathEscape(s.TaskID), map[string]any{"state": "done", "version": 1}, nil, 404, nil)
	c.add("tasks.stale_version_rejected", err == nil)
	var createdComment map[string]string
	err = c.call("POST", "/api/v1/tasks/"+url.PathEscape(s.TaskID)+"/comments", map[string]string{"body": "trusted acceptance comment"}, nil, 201, &createdComment)
	s.CommentID = createdComment["id"]
	c.add("comments.create", err == nil && taskFlowUUID.MatchString(s.CommentID))

	// Owner assigns viewer; viewer may read, but cannot mutate.
	c.cookies = cloneCookies(aCookies)
	err = c.call("PUT", "/api/v1/workspaces/"+url.PathEscape(s.WorkspaceID)+"/members/"+url.PathEscape(bEmail), map[string]string{"role": "viewer"}, nil, 200, nil)
	c.add("rbac.owner_assigns_viewer", err == nil)
	bCookies := cloneCookies(c.cookies)
	// Re-registering is unnecessary; login B obtains a fresh checked server cookie.
	c.cookies = map[string]string{}
	var login map[string]string
	err = c.call("POST", "/api/v1/sessions", map[string]string{"email": bEmail, "password": bPassword}, nil, 200, &login)
	viewerCookies := cloneCookies(c.cookies)
	var viewerTasks taskFlowPage[taskFlowTask]
	listErr := c.call("GET", "/api/v1/projects/"+url.PathEscape(s.ProjectID)+"/tasks", nil, nil, 200, &viewerTasks)
	c.add("rbac.viewer_can_read", err == nil && listErr == nil && len(viewerTasks.Items) == 1)
	err = c.call("PATCH", "/api/v1/tasks/"+url.PathEscape(s.TaskID), map[string]any{"state": "done", "version": 2}, nil, 404, nil)
	c.add("rbac.viewer_task_update_denied", err == nil)
	err = c.call("POST", "/api/v1/tasks/"+url.PathEscape(s.TaskID)+"/comments", map[string]string{"body": "forbidden viewer comment"}, nil, 404, nil)
	c.add("rbac.viewer_comment_denied", err == nil)
	err = c.call("PUT", "/api/v1/workspaces/"+url.PathEscape(s.WorkspaceID)+"/members/"+url.PathEscape(s.Email), map[string]string{"role": "viewer"}, nil, 404, nil)
	c.add("rbac.non_owner_membership_change_denied", err == nil)
	_ = bCookies

	c.cookies = cloneCookies(aCookies)
	err = c.call("PUT", "/api/v1/workspaces/"+url.PathEscape(s.WorkspaceID)+"/members/"+url.PathEscape(bEmail), map[string]string{"role": "editor"}, nil, 200, nil)
	c.add("rbac.owner_promotes_editor", err == nil)
	c.cookies = viewerCookies
	err = c.call("POST", "/api/v1/tasks/"+url.PathEscape(s.TaskID)+"/comments", map[string]string{"body": "editor comment"}, nil, 201, &createdComment)
	c.add("rbac.editor_comment_allowed", err == nil && taskFlowUUID.MatchString(createdComment["id"]))
	c.cookies = cloneCookies(aCookies)
	var comments taskFlowPage[taskFlowComment]
	err = c.call("GET", "/api/v1/tasks/"+url.PathEscape(s.TaskID)+"/comments", nil, nil, 200, &comments)
	commentFound := false
	for _, comment := range comments.Items {
		if comment.ID == s.CommentID && comment.Body == "trusted acceptance comment" {
			commentFound = true
		}
	}
	c.add("comments.list", err == nil && commentFound && len(comments.Items) >= 1 && len(comments.Items) <= 100)
	if !registered {
		c.add("auth.register_required", false)
	}

	err = c.call("DELETE", "/api/v1/sessions/current", nil, nil, 204, nil)
	c.add("auth.logout_revoke", err == nil)
	err = c.call("GET", "/api/v1/workspaces", nil, nil, 401, nil)
	c.add("auth.revoked_session_rejected", err == nil)
	return s
}

func (c *checker) taskFlowPersistence(raw json.RawMessage) any {
	var s taskFlowState
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if len(raw) == 0 || d.Decode(&s) != nil || !taskFlowUUID.MatchString(s.WorkspaceID) || !taskFlowUUID.MatchString(s.ProjectID) || !taskFlowUUID.MatchString(s.SecondProjectID) || !taskFlowUUID.MatchString(s.TaskID) || !taskFlowUUID.MatchString(s.CommentID) || !emailPattern.MatchString(s.Email) || len(s.Password) < 12 || len(s.IdempotencyKey) < 1 || len(s.IdempotencyKey) > 128 {
		c.add("persistence.state", false)
		return nil
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		c.add("persistence.state", false)
		return nil
	}
	c.add("persistence.state", true)
	var login map[string]string
	err := c.call("POST", "/api/v1/sessions", map[string]string{"email": s.Email, "password": s.Password}, nil, 200, &login)
	c.add("persistence.login", err == nil && login["status"] == "ok")
	var workspaces taskFlowPage[taskFlowWorkspace]
	err = c.call("GET", "/api/v1/workspaces", nil, nil, 200, &workspaces)
	found := false
	for _, w := range workspaces.Items {
		if w.ID == s.WorkspaceID && w.Role == "owner" {
			found = true
		}
	}
	c.add("persistence.workspace", err == nil && found)
	var projects taskFlowPage[taskFlowProject]
	err = c.call("GET", "/api/v1/workspaces/"+url.PathEscape(s.WorkspaceID)+"/projects", nil, nil, 200, &projects)
	projectFound, secondFound := false, false
	for _, p := range projects.Items {
		projectFound = projectFound || p.ID == s.ProjectID
		secondFound = secondFound || p.ID == s.SecondProjectID
	}
	c.add("persistence.projects", err == nil && projectFound && secondFound)
	var tasks taskFlowPage[taskFlowTask]
	err = c.call("GET", "/api/v1/projects/"+url.PathEscape(s.ProjectID)+"/tasks", nil, nil, 200, &tasks)
	taskFound := false
	for _, task := range tasks.Items {
		if task.ID == s.TaskID && task.State == "doing" && task.Version == 2 {
			taskFound = true
		}
	}
	c.add("persistence.task", err == nil && taskFound)
	var comments taskFlowPage[taskFlowComment]
	err = c.call("GET", "/api/v1/tasks/"+url.PathEscape(s.TaskID)+"/comments", nil, nil, 200, &comments)
	commentFound := false
	for _, comment := range comments.Items {
		if comment.ID == s.CommentID && comment.Body == "trusted acceptance comment" {
			commentFound = true
		}
	}
	c.add("persistence.comment", err == nil && commentFound)
	return nil
}
