package main

func CriterionState(target, source string) string { if target == "" || source == "" { return "incomplete" }; return "review" }
