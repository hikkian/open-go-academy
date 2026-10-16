package main

func RoleCan(role, action string) bool {
	switch role {
	case "owner":
		return action == "read" || action == "write" || action == "delete"
	case "editor":
		return action == "read" || action == "write"
	case "viewer":
		return action == "read"
	default:
		return false
	}
}
