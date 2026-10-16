package main

func RollbackAction(dataCompatible, incidentOngoing bool) string {
	if !incidentOngoing {
		return "observe"
	}
	if dataCompatible {
		return "rollback_code"
	}
	return "forward_fix_review"
}
