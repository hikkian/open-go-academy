package main

func DeletionState(active, pendingCopies bool) string { if active { return "active" }; if pendingCopies { return "cleanup-pending" }; return "deleted" }
