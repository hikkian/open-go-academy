package main

func ApplyVersionedEdits(v []int) []int {
	if len(v) < 2 || v[1] < 0 {
		return []int{0, 0, 0}
	}
	value, ver, count := v[0], v[1], 0
	for i := 2; i+1 < len(v); i += 2 {
		expected, next := v[i], v[i+1]
		if expected == ver && ver < int(^uint(0)>>1) {
			value = next
			ver++
			count++
		}
	}
	return []int{value, ver, count}
}
