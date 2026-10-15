package main

func OptionalValue(present bool, value string) (string, bool) { if !present { return "", false }; return value, true }
