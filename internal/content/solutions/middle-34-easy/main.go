package main

func RetryDecision(attempt, maxAttempts int, transient bool) string {
	if attempt < 0 || maxAttempts <= 0 {
		return "invalid"
	}
	if !transient {
		return "stop"
	}
	if attempt >= maxAttempts {
		return "exhausted"
	}
	return "retry"
}
