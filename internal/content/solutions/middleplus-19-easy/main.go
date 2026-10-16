package main

func FaultOutcome(stage string, committed bool, observed bool) string {
	switch stage {
	case "before_commit":
		if committed {
			return "inconsistent"
		}
		return "no_change"
	case "after_commit":
		if !committed {
			return "inconsistent"
		}
		if observed {
			return "recover"
		}
		return "reconcile"
	case "publish":
		if !committed {
			return "no_change"
		}
		if observed {
			return "retry"
		}
		return "reconcile"
	default:
		return "unknown"
	}
}
