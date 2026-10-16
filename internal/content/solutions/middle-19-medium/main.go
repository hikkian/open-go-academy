package main

func RetryActionMedium(attempt, maxAttempts int64, failed, temporary bool) string {
	if attempt <= 0 || maxAttempts <= 0 { return "invalid" }; if !failed { return "ack" }; if temporary && attempt < maxAttempts { return "retry" }; return "dead_letter"
}
