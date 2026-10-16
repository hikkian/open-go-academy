package main

func BackupState(exists, verified bool) string { if !exists { return "missing" }; if !verified { return "unverified" }; return "candidate" }
