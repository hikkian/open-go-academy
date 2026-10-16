package main

func ValidHeapProfile(inuseBytes, allocBytes int64) bool { return inuseBytes >= 0 && allocBytes >= 0 }
