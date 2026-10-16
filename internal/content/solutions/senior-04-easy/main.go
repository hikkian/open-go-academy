package main

func CommandState(key, previous string) string { if key == "" { return "missing-key" }; if key == previous { return "replay" }; return "new" }
