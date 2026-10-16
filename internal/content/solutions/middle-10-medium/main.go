package main

func LeastLoadedWorker(v []int) []int {
	if len(v) < 1 || v[0] <= 0 {
		return []int{}
	}
	loads := make([]int, v[0])
	max := int(^uint(0) >> 1)
	o := []int{}
	for _, d := range v[1:] {
		if d < 0 {
			return []int{}
		}
		best := 0
		for i := 1; i < len(loads); i++ {
			if loads[i] < loads[best] {
				best = i
			}
		}
		o = append(o, best)
		if d > max-loads[best] {
			loads[best] = max
		} else {
			loads[best] += d
		}
	}
	return o
}
