package main

func TokenDecisionEasy(available, capacity, cost int64) string {
	if capacity < 0 || available < 0 || available > capacity || cost <= 0 { return "invalid" }; if available >= cost { return "allow" }; return "deny"
}
