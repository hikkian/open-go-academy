package main

func LoadMode(offered int, capacity int) string {
	if offered < 0 || capacity <= 0 {
		return "invalid"
	}
	if offered <= capacity {
		return "normal"
	}
	return "overloaded"
}
