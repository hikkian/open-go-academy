package main

func classify(n int) string { if n < 0 { return "мороз" }; if n < 20 { return "норма" }; if n < 35 { return "тепло" }; return "жара" }
