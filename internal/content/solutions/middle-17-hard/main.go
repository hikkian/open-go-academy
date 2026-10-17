package main

func OutboxCrashHard(committed bool, eventID string, publish func(string) bool, markPublished func(string) bool) string {
	if !committed { return "discard" }
	if !publish(eventID) { return "retry_publish" }
	if !markPublished(eventID) { return "duplicate_risk" }
	return "complete"
}
