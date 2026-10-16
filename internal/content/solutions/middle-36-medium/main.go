package main

func TaskTransition(from, to string) bool {
	switch from {
	case "open":
		return to == "in_progress" || to == "cancelled"
	case "in_progress":
		return to == "done" || to == "cancelled"
	default:
		return false
	}
}
