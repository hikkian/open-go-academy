package main

func RiskAction(highImpact, controlTested bool) string { if highImpact && !controlTested { return "mitigate-first" }; if !controlTested { return "verify-control" }; return "review-residual" }
