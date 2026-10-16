package main

func Spare(capacity, demand int64) (margin int64, valid bool) { if capacity < 0 || demand < 0 { return 0, false }; return capacity - demand, true }
