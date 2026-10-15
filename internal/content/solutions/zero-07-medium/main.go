package main

func countAbove(values []int, threshold int) int { count := 0; for _, value := range values { if value > threshold { count++ } }; return count }
