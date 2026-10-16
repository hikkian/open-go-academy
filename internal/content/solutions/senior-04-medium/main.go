package main

func VersionState(expected, current int64) string { if expected < 0 || current < 0 { return "invalid" }; if expected != current { return "conflict" }; return "apply" }
