package main

func appendCopy(values []int, value int) []int { result := make([]int, len(values)); copy(result, values); return append(result, value) }
