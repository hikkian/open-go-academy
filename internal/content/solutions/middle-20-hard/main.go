package main

func PublishOutcomeHard(mandatory, brokerConfirm, returned bool) string {
	if returned { return "unrouted" }; if !brokerConfirm { return "unknown" }; return "confirmed"
}
