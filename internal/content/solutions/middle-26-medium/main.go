package main

type TransactionMedium interface {
	Begin() bool
	Apply() string
	Commit() bool
	Rollback()
}

func RunTransactionMedium(valid bool, tx TransactionMedium) string {
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
		tx.Rollback()
		return "commit_outcome_unknown"
	}
	return "committed"
}
