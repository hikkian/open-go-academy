package main

func ProjectionState(source, view int64) string { if source < 0 || view < 0 { return "invalid" }; if source == view { return "caught-up" }; if source > view { return "lagging" }; return "unexpected-ahead" }
