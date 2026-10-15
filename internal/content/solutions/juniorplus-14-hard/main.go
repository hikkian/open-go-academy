package main

func TransactionTestOutcome(failed, rollback, cleaned bool) string {
	if !cleaned {
		return "cleanup_failed"
	}
	if failed {
		if rollback {
			return "rollback_verified"
		}
		return "rollback_missing"
	}
	if rollback {
		return "unexpected_rollback"
	}
	return "committed"
}
