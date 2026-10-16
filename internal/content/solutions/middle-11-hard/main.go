package main

func SerializationRetryBudget(v []int) []int {
	if len(v) != 3 || v[0] < 0 || v[1] < 0 || v[2] < 0 {
		return []int{0}
	}
	if v[2] == 0 {
		return []int{0}
	}
	left := v[0] - v[1]
	if left < 0 {
		left = 0
	}
	return []int{left}
}
