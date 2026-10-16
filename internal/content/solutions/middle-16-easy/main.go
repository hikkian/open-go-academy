package main

func DeliveryModeEasy(mode string) string {
	switch mode { case "loss_ok": return "at_most_once"; case "retry_possible": return "at_least_once"; default: return "invalid" }
}
