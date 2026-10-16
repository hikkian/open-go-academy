package main

func BudgetRemaining(total int, failed int, maxErrorPercent int) bool {
	if total <= 0 || failed < 0 || failed > total || maxErrorPercent < 0 || maxErrorPercent > 100 {
		return false
	}
	// Compute floor(total*percent/100) without forming either overflowing product.
	allowed := (total/100)*maxErrorPercent + (total%100)*maxErrorPercent/100
	return failed <= allowed
}
