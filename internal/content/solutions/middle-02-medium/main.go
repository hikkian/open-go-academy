package main

func JoinCompleted(v []int) []int {
	o := []int{}
	for i := 0; i+1 < len(v); i += 2 {
		if v[i+1] == 1 {
			o = append(o, v[i])
		}
	}
	return o
}
