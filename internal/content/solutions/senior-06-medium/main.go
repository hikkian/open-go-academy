package main

func ChangePath(ownerKnown, bypassExists bool) string { if !ownerKnown { return "clarify-owner" }; if bypassExists { return "close-bypass" }; return "owned-path" }
