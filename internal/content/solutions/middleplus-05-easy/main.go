package main

func MemoryBudget(used, limit int64) bool { return used >= 0 && limit > 0 && used <= limit }
