package main

func OwnsResource(p, o int64) bool { return p > 0 && o > 0 && p == o }
