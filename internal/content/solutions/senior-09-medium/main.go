package main

func LossState(lossSeconds, allowedSeconds int64) string { if lossSeconds < 0 || allowedSeconds < 0 { return "invalid" }; if lossSeconds <= allowedSeconds { return "within-target" }; return "exceeds-target" }
