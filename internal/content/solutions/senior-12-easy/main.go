package main

func SecretState(configured bool) string { if configured { return "present" }; return "missing" }
