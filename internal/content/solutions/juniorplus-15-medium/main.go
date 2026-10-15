package main

import "time"

func SessionExpired(now, exp time.Time) bool { return !exp.After(now) }
