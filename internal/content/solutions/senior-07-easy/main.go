package main

func ReadDecision(required, visible int64) string { if required < 0 || visible < 0 { return "invalid" }; if visible >= required { return "serve" }; return "wait-or-route" }
