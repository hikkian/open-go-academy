package main

func Response(freshnessRequired, dependencyAvailable bool) string { if dependencyAvailable { return "read-authority" }; if freshnessRequired { return "fail-closed" }; return "stale-with-label" }
