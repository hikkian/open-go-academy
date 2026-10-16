package main

func WorkerFair(arrival []int, wait []int, selected []bool, maxWait int) bool {
	if len(arrival) == 0 || len(arrival) != len(wait) || len(arrival) != len(selected) || maxWait < 0 {
		return false
	}
	for i := range arrival {
		if arrival[i] < 0 || wait[i] < 0 {
			return false
		}
		if !selected[i] && wait[i] >= maxWait {
			return false
		}
	}
	return true
}
