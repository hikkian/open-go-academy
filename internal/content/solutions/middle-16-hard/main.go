package main

func DeliveryWindowHard(eventID string, alreadyProcessed bool, apply func(string) error, ack func(string) error) string {
	if !alreadyProcessed {
		if err := apply(eventID); err != nil { return "effect_failed" }
	}
	if err := ack(eventID); err != nil { return "effect_committed_ack_unknown" }
	return "acknowledged"
}
