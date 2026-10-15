package main

func countPassing(scores []int) int { count := 0; for _, score := range scores { if score >= 60 { count++ } }; return count }
