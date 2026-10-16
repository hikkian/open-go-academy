package main

func WorkerDisposition(attempt, maxAttempts int, transient, effectApplied bool) string {
	if attempt < 0 || maxAttempts <= 0 {
		return "invalid"
	}
	if effectApplied {
		return "ack"
	}
	if !transient {
		return "dead_letter"
	}
	if attempt >= maxAttempts {
		return "dead_letter"
	}
	return "retry"
}
