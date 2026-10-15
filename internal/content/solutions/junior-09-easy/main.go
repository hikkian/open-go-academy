package main

func maxValue(values []int) (int, bool) {
	if len(values) == 0 {
		return 0, false
	}
	m := values[0]
	for _, v := range values[1:] {
		if v > m {
			m = v
		}
	}
	return m, true
}
