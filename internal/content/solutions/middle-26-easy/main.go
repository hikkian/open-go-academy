package main

type BeginnerEasy interface {
	Begin() bool
}

func ValidationBoundaryEasy(valid bool, tx BeginnerEasy) string {
	if !valid {
		return "invalid"
	}
	if tx == nil {
		return "dependency_missing"
	}
	if !tx.Begin() {
		return "begin_failed"
	}
	return "ready_to_write"
}
