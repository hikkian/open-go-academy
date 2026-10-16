package main

func OutboxAttempt(state string, event string, attempts int, maxAttempts int) string {
	if attempts < 0 || maxAttempts <= 0 || attempts > maxAttempts {
		return "invalid"
	}
	if state == "pending" && event == "publish" {
		if attempts >= maxAttempts {
			return "dead_letter"
		}
		return "delivered"
	}
	if (state == "pending" && event == "retry") || (state == "delivered" && event == "crash") {
		if attempts >= maxAttempts {
			return "dead_letter"
		}
		return "pending"
	}
	return OutboxTransition(state, event)
}

func OutboxTransition(state string, event string) string {
	if state == "absent" && event == "persist" {
		return "pending"
	}
	if state == "absent" && event == "rollback" {
		return "absent"
	}
	if state == "pending" && event == "publish" {
		return "delivered"
	}
	if state == "pending" && event == "crash" {
		return "pending"
	}
	if state == "delivered" && event == "ack" {
		return "acked"
	}
	if state == "delivered" && event == "crash" {
		return "pending"
	}
	if state == "acked" && event == "publish" {
		return "acked"
	}
	return "invalid"
}
