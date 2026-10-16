package main

func Cutover(restoreVerified, oldWriterFenced bool) string { if restoreVerified && oldWriterFenced { return "eligible" }; return "hold" }
