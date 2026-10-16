package main

func TradeoffRecorded(benefit, cost, revisitTrigger string) bool {
	return benefit != "" && cost != "" && revisitTrigger != ""
}
