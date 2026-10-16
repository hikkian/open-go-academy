package main

func RollbackState(reversible, newWrites bool) string { if reversible { return "rollback" }; if newWrites { return "roll-forward" }; return "review" }
