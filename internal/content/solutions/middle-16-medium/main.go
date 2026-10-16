package main

func DeliveryWindowMedium(published bool, handlerCommitted bool, acked bool) string {
	if acked && !handlerCommitted { return "unsafe_ack" }
	if !published { return "not_delivered" }
	if handlerCommitted && !acked { return "redelivery_possible" }
	if handlerCommitted && acked { return "complete" }
	return "pending"
}
