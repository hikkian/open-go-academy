package main

func BoundaryAction(authenticated, authorized bool) string { if !authenticated { return "authenticate" }; if !authorized { return "deny" }; return "continue" }
