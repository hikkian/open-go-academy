package main

func RequirementKind(answer string) string { if answer == "" { return "clarify" }; if answer == "confirmed" { return "evidence" }; return "assumption" }
