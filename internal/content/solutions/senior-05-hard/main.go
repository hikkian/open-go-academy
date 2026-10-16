package main

func VersionPolicy(known, invariantCritical bool) string { if known { return "process" }; if invariantCritical { return "quarantine" }; return "review" }
