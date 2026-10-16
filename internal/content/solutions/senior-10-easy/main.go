package main

func AssetClass(secret, userData bool) string { if secret { return "restricted" }; if userData { return "sensitive" }; return "general" }
