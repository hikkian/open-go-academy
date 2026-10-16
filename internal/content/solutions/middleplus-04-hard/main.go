package main

func HotPathCandidate(flat, cumulative, total, minPercent int) bool {
	if total <= 0 || flat < 0 || cumulative < flat || cumulative > total || minPercent < 0 || minPercent > 100 {
		return false
	}
	threshold := (total / 100) * minPercent
	rem := (total % 100) * minPercent
	threshold += (rem + 99) / 100
	return flat >= threshold
}
