package main

func FaultStage(stage string) string {
	switch stage {
	case "before_commit":
		return "rollback"
	case "after_commit":
		return "reconcile"
	case "publish":
		return "republish_idempotently"
	default:
		return "unknown"
	}
}
