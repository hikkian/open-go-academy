package main

func IndexCandidate(matches, total, estimatedRows int) bool {
	if total <= 0 || matches < 0 || matches > total || estimatedRows < 0 {
		return false
	}
	half := total / 2
	if total%2 > 0 {
		half++
	}
	return estimatedRows < half
}
