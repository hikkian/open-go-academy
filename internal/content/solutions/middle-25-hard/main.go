package main

type ProfileCreatorHard interface {
	CreateProfile(name string) bool
}

type AuditSinkHard interface {
	Record(event string) bool
}

func CreateProfileHard(repository ProfileCreatorHard, audit AuditSinkHard, name string) string {
	if name == "" {
		return "invalid_request"
	}
	if repository == nil || audit == nil {
		return "dependency_missing"
	}
	if !repository.CreateProfile(name) {
		return "create_error"
	}
	if !audit.Record("profile_created") {
		return "audit_error_after_create"
	}
	return "created"
}
