package main

func DeadlineState(remainingMillis int64) string { if remainingMillis < 0 { return "invalid" }; if remainingMillis == 0 { return "expired" }; return "continue" }
