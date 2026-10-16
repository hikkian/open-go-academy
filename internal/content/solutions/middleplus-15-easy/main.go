package main

func QueueBand(depth int, capacity int) string {
	if depth < 0 || capacity <= 0 {
		return "invalid"
	}
	if depth == 0 {
		return "empty"
	}
	if depth >= capacity {
		return "full"
	}
	return "queued"
}
