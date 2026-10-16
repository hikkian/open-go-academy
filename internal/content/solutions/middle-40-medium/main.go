package main

func AcceptanceStatus(required, passed int, openCriticalRisk bool) string {
	if required <= 0 || passed < 0 || passed > required {
		return "invalid"
	}
	if openCriticalRisk {
		return "hold_risk"
	}
	if passed < required {
		return "incomplete"
	}
	return "ready_for_review"
}
