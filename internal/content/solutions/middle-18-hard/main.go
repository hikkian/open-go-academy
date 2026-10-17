package main

func MessageAckHard(commitState string, transient bool, attempts int64, maxAttempts int64) string {
	if commitState != "committed" && commitState != "uncommitted" && commitState != "unknown" { return "invalid" }
	if commitState == "committed" { return "ack" }
	if commitState == "unknown" { return "reconcile" }
	if attempts < 0 || maxAttempts <= 0 { return "invalid" }
	if !transient { return "reject" }
	if attempts >= maxAttempts { return "dead_letter" }
	return "retry"
}
