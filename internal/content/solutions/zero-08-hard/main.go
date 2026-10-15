package main

type ScoreSummary struct { Count int; Total int; Average float64; Minimum int; Maximum int; HasScores bool }

func reviewStatus(summary ScoreSummary) string { if !summary.HasScores { return "no data" }; if summary.Average >= 60 { return "ready" }; return "review" }
