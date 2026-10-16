package main

func BudgetAction(total, failed, maxErrorPercent int) string {
	if total <= 0 || failed < 0 || failed > total || maxErrorPercent < 0 || maxErrorPercent > 100 {
		return "insufficient_data"
	}
	if failed > allowedFailures(total, maxErrorPercent) {
		return "freeze_risky_changes"
	}
	return "within_budget"
}

// allowedFailures returns floor(total*percent/100) without overflowing int.
func allowedFailures(total, percent int) int {
	quotient, remainder := total/100, total%100
	return quotient*percent + remainder*percent/100
}
