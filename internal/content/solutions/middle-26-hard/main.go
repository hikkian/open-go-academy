package main

func TransactionOutcomeHard(valid, conflict, writeFailed, commitFailed bool) string {
	if !valid { return "invalid" }; if conflict || writeFailed || commitFailed { return "rollback" }; return "commit_requested"
}
