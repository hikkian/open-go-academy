package main

func LogFieldAllowed(key string) bool {
	switch key {
	case "request_id", "route", "status", "duration_ms":
		return true
	default:
		return false
	}
}
