package main

func ShouldRollback(operationFailed, committed bool) bool { return operationFailed && !committed }
