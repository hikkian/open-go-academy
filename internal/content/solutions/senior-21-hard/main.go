package main

func CostDecision(estimateValid, workloadMeasured, budgetOwner bool) bool {
	return estimateValid && workloadMeasured && budgetOwner
}
