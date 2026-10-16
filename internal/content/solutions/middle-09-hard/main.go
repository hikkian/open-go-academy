package main

func UnblockPlan(v []int) []int {
	if len(v) != 3 || v[0] < 0 || v[1] < 0 || v[2] < 0 {
		return []int{0, 0}
	}
	receivers := v[1] - v[0]
	if receivers < 0 {
		receivers = 0
	}
	paired := v[2]
	if paired > receivers {
		paired = receivers
	}
	return []int{v[2] - paired, receivers - paired}
}
