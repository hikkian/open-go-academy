package main

func SLOMeets(total, failed, maxErrorPercent int) bool {
	if total <= 0 || failed < 0 || failed > total || maxErrorPercent < 0 || maxErrorPercent > 100 {
		return false
	}
	return failed <= allowedFailures(total, maxErrorPercent)
}

// allowedFailures returns floor(total*percent/100) without overflowing int.
func allowedFailures(total, percent int) int {
	quotient, remainder := total/100, total%100
	return quotient*percent + remainder*percent/100
}
