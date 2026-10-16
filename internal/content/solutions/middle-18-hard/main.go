package main

func MessageAckHard(transient bool, attempts int64, handlerCommitted bool, maxAttempts int64) string {
	if attempts < 0 || maxAttempts <= 0 { return "invalid" }
	if handlerCommitted { return "ack" }
	if !transient { return "reject" }
	if attempts >= maxAttempts { return "dead_letter" }
	return "retry"
}
