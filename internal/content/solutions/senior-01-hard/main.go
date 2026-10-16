package main

func NextStep(reversible, highImpact bool) string { if !reversible && highImpact { return "evidence-first" }; if reversible { return "timeboxed-trial" }; return "owner-review" }
