package main

func LoadDecision(offered int, capacity int, critical bool) string {
	if offered < 0 || capacity <= 0 {
		return "invalid"
	}
	if offered <= capacity {
		return "accept"
	}
	if critical {
		return "prioritize"
	}
	return "shed"
}
