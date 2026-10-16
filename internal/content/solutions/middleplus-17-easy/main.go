package main

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
