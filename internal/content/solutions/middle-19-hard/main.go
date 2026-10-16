package main

func RetryTransitionHard(attempt, maxAttempts int64, failed, temporary, acknowledged bool) string {
	if attempt <= 0 || maxAttempts <= 0 { return "invalid" }; if acknowledged || !failed { return "ack" }; if temporary && attempt < maxAttempts { return "retry" }; return "dead_letter"
}
