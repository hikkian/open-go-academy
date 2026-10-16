package main

func CloseStatus(active int, graceExpired bool) string {
	if active < 0 {
		return "invalid"
	}
	if active == 0 {
		return "closed"
	}
	if graceExpired {
		return "cancel_remaining"
	}
	return "draining"
}
