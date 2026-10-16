package main

func CircuitAction(state string, deadlineLeftMillis, active, limit int) string {
	if deadlineLeftMillis <= 0 {
		return "timeout"
	}
	if active < 0 || limit <= 0 {
		return "invalid"
	}
	if active >= limit {
		return "bulkhead_full"
	}
	switch state {
	case "open":
		return "reject_open"
	case "half_open":
		return "probe"
	case "closed":
		return "call"
	default:
		return "invalid_state"
	}
}
