package main

func AdmissionDecision(depth int, limit int, waitMillis int, deadlineMillis int, cancelled bool, shed bool) string {
	if depth < 0 || limit <= 0 || waitMillis < 0 || deadlineMillis < 0 {
		return "invalid"
	}
	if cancelled {
		return "cancelled"
	}
	if depth < limit {
		return "admit"
	}
	if shed {
		return "shed"
	}
	if deadlineMillis == 0 || waitMillis >= deadlineMillis {
		return "deadline"
	}
	if waitMillis == 0 {
		return "reject"
	}
	return "wait"
}
