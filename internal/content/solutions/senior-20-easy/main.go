package main

func ReleaseGate(testsPassed, migrationReversible, rollbackReady bool) string {
	if !testsPassed {
		return "hold_tests"
	}
	if !migrationReversible {
		return "hold_migration"
	}
	if !rollbackReady {
		return "hold_rollback"
	}
	return "eligible_for_review"
}
