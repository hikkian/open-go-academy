package main

func MonthlyEstimate(units, unitCost int64) int64 {
	if units < 0 || unitCost < 0 {
		return -1
	}
	if units != 0 && unitCost > 0 && units > 9223372036854775807/unitCost {
		return -1
	}
	return units * unitCost
}
