package main

func IdempotencyStateMedium(found bool, payloadMatches bool, completed bool) string {
	if !found { return "new" }
	if !payloadMatches { return "conflict" }
	if !completed { return "in_progress" }
	return "replay"
}
