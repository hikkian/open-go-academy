package main

func PageLimit(requested int) int { if requested < 1 { return 20 }; if requested > 100 { return 100 }; return requested }
