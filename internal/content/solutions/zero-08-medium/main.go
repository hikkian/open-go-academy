package main

type ScoreSummary struct { Count int; Total int; Average float64; Minimum int; Maximum int; HasScores bool }

func summarizeScores(scores []int) ScoreSummary {
 out := ScoreSummary{Count: len(scores)}
 if len(scores) == 0 { return out }
 out.HasScores = true; out.Minimum = scores[0]; out.Maximum = scores[0]
 for _, score := range scores { out.Total += score; if score < out.Minimum { out.Minimum = score }; if score > out.Maximum { out.Maximum = score } }
 out.Average = float64(out.Total) / float64(out.Count)
 return out
}
