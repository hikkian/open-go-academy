package main

func IdempotencyRaceHard(inserted bool, payloadMatches bool, completed bool, hasStoredResponse bool) string {
	if inserted { return "created" }
	if !payloadMatches { return "conflict" }
	if !completed { return "in_progress" }
	if hasStoredResponse { return "replay" }
	return "unknown_result"
}
