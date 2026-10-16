package main

func RecoveryStep(stage string, committed bool, observed bool, recovered bool, retryable bool) string {
	if recovered {
		return "done"
	}
	switch stage {
	case "before_commit":
		if committed || observed {
			return "inspect"
		}
		return "rollback"
	case "after_commit":
		if !committed {
			return "inspect"
		}
		if !observed {
			return "reconcile"
		}
		return "inspect"
	case "publish":
		if !committed {
			return "inspect"
		}
		if !observed {
			return "reconcile"
		}
		if retryable {
			return "retry"
		}
		return "dead_letter"
	default:
		return "inspect"
	}
}
