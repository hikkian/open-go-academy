package main

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	b := []byte{}
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func WorkerBatch(arrival []int, wait []int, workers int, maxWait int) string {
	if len(arrival) == 0 || len(arrival) != len(wait) || workers < 0 || maxWait < 0 {
		return "invalid"
	}
	for i := range arrival {
		if arrival[i] < 0 || wait[i] < 0 {
			return "invalid"
		}
	}
	if workers > len(arrival) {
		workers = len(arrival)
	}
	chosen := make([]bool, len(arrival))
	out := ""
	for n := 0; n < workers; n++ {
		best := -1
		for i := range arrival {
			if chosen[i] || wait[i] < maxWait {
				continue
			}
			if best < 0 || wait[i] > wait[best] || (wait[i] == wait[best] && arrival[i] < arrival[best]) {
				best = i
			}
		}
		if best < 0 {
			for i := range arrival {
				if !chosen[i] && (best < 0 || arrival[i] < arrival[best]) {
					best = i
				}
			}
		}
		chosen[best] = true
		if n > 0 {
			out += ","
		}
		out += itoa(best)
	}
	return out
}
