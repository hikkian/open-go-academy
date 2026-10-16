package main

func MessageDispositionEasy(transient bool, attempts int64, succeeded bool) string {
	if attempts < 0 { return "invalid" }
	if succeeded { return "ack" }
	if !transient { return "reject" }
	if attempts >= 3 { return "dead_letter" }
	return "retry"
}
