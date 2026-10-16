package main

func OutboxCrashHard(committed bool, brokerAccepted bool, rowMarked bool) string {
	if !committed { return "discard" }
	if !brokerAccepted { return "publish" }
	if !rowMarked { return "retry_may_duplicate" }
	return "complete"
}
