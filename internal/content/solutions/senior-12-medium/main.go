package main

func Rotation(oldActive, newVerified bool) string { if !newVerified { return "hold" }; if oldActive { return "switch" }; return "complete" }
