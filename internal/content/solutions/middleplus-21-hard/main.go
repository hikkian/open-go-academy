package main

func BudgetRemaining(total int, failed int, maxErrorPercent int) int {
	minInt := -int(^uint(0)>>1) - 1
	if total <= 0 || failed < 0 || failed > total || maxErrorPercent < 0 || maxErrorPercent > 100 {
		return minInt
	}
	// Compute floor(total*percent/100) without forming an overflowing product.
	allowed := (total/100)*maxErrorPercent + (total%100)*maxErrorPercent/100
	return allowed - failed
}
