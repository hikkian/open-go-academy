package main

func CapacityBand(demand, testedCapacity int) string {
	if demand < 0 || testedCapacity <= 0 {
		return "invalid"
	}
	if demand > testedCapacity {
		return "exceeds_test"
	}
	if demand >= testedCapacity-testedCapacity/5 {
		return "near_tested_limit"
	}
	return "within_tested_range"
}
