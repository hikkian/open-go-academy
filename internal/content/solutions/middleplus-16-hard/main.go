package main

func RetryDecision(attempt int, maxAttempts int, budgetRemaining int, delayMillis int, deadlineRemainingMillis int, transient bool) string {
	if attempt < 0 || maxAttempts <= 0 || budgetRemaining < 0 || delayMillis < 0 || deadlineRemainingMillis < 0 {
		return "invalid"
	}
	if !transient {
		return "stop"
	}
	if attempt >= maxAttempts {
		return "exhausted"
	}
	if budgetRemaining == 0 {
		return "budget_exhausted"
	}
	if deadlineRemainingMillis == 0 || delayMillis >= deadlineRemainingMillis {
		return "deadline"
	}
	return "retry"
}
