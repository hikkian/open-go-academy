package main

func SelectivityClass(matches, total int) string {
	if total <= 0 || matches < 0 || matches > total {
		return "invalid"
	}
	lowCeil := total / 10
	if total%10 > 0 {
		lowCeil++
	}
	if matches < lowCeil {
		return "low"
	}
	half := total / 2
	if total%2 > 0 {
		half++
	}
	if matches < half {
		return "medium"
	}
	return "high"
}
