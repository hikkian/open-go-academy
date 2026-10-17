package main

type TransactionHard interface {
	Begin() bool
	Apply() string
	Commit() bool
	Rollback()
}

func RunTransactionHard(valid bool, tx TransactionHard) string {
	if !valid {
		return "invalid"
	}
	if tx == nil {
		return "dependency_missing"
	}
	if !tx.Begin() {
		return "begin_failed"
	}
	switch tx.Apply() {
	case "ok":
	case "conflict":
		tx.Rollback()
		return "conflict"
	case "write_failed":
		tx.Rollback()
		return "write_error"
	default:
		tx.Rollback()
		return "internal_error"
	}
	if !tx.Commit() {
		// Commit outcome may be unknown; this is a best-effort cleanup attempt.
		tx.Rollback()
		return "commit_outcome_unknown"
	}
	return "committed"
}
