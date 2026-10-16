package main

func MessageAckMedium(transient bool, attempts int64, handlerCommitted bool) string {
	if attempts < 0 { return "invalid" }
	if handlerCommitted { return "ack" }
	if !transient { return "reject" }
	if attempts >= 3 { return "dead_letter" }
	return "retry"
}
