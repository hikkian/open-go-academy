package main

func IncidentAction(impact int, ongoing bool, mitigationReady bool, evidenceCount int, ownerAssigned bool, dueDay int) string {
	if impact < 0 || evidenceCount < 0 || dueDay < 0 {
		return "invalid"
	}
	if ongoing {
		return "mitigate"
	}
	if evidenceCount < 2 {
		return "collect_evidence"
	}
	if mitigationReady {
		return "verify"
	}
	if ownerAssigned && dueDay > 0 {
		return "track_action"
	}
	return "assign_action"
}
