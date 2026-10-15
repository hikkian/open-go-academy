package main
func safeAt(values []int, index int) (int, bool) { if index < 0 || index >= len(values) { return 0, false }; return values[index], true }
