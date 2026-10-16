package main

func IdempotencyOutcomeEasy(seen bool, samePayload bool) string {
	if !seen { return "new" }
	if samePayload { return "replay" }
	return "conflict"
}
