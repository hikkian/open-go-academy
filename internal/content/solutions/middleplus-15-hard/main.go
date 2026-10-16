package main

func QueueAction(depth int, capacity int, oldestSeconds int, maxAgeSeconds int, consumerPaused bool) string {
	if depth < 0 || capacity <= 0 || oldestSeconds < 0 || maxAgeSeconds <= 0 {
		return "invalid"
	}
	if consumerPaused {
		return "paused"
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
