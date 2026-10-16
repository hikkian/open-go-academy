package main

func WindowState(windowSeconds, samples int64) string { if windowSeconds <= 0 || samples < 0 { return "invalid" }; if samples == 0 { return "no-data" }; return "review" }
