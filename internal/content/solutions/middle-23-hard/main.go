package main

func ProjectionEventActionHard(current, incoming int64, deleted, alreadyApplied bool) string {
	if current < 0 || incoming < 0 { return "invalid" }; if alreadyApplied { return "duplicate" }; if incoming <= current { return "ignore" }; if deleted { return "invalidate" }; return "refresh"
}
