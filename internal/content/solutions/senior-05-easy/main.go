package main

func Boundary(independentRelease, networkNeeded bool) string { if independentRelease && networkNeeded { return "service-candidate" }; return "module-first" }
