package main

func IncidentAction(ongoing bool, impact int, mitigationReady bool) string {
	if impact < 0 {
		return "invalid"
	}
	if ongoing {
		return "mitigate"
	}
	if !mitigationReady {
		return "assign_action"
	}
	return "verify"
}
