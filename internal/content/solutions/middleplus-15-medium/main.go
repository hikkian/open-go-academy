package main

func QueuePressure(depth int, capacity int, oldestSeconds int, maxAgeSeconds int) string {
	if depth < 0 || capacity <= 0 || oldestSeconds < 0 || maxAgeSeconds <= 0 {
		return "invalid"
	}
	if depth == 0 {
		return "empty"
	}
	if depth >= capacity {
		return "full"
	}
	if oldestSeconds >= maxAgeSeconds {
		return "stale"
	}
	return "normal"
}
