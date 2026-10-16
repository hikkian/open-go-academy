package main

func PerSecond(events, seconds int64) int64 { if events < 0 || seconds <= 0 { return -1 }; return events / seconds }
