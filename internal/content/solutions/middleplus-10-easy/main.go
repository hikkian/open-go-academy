package main

func ReadSource(lagSeconds, maxLag int) string {
	if lagSeconds < 0 || maxLag < 0 {
		return "invalid"
	}
	if lagSeconds <= maxLag {
		return "replica"
	}
	return "primary"
}
