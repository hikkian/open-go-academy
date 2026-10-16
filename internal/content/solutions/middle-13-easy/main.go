package main

func MatchSelectivity(matches, total int64) string {
	if matches < 0 || total < 0 || matches > total {
		return "invalid"
	}
	if total == 0 {
		return "no_rows"
	}
	if matches == 0 {
		return "none"
	}
	if matches <= total/100 {
		return "sparse"
	}
	return "common"
}
