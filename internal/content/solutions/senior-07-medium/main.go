package main

func Delivery(seen, validID bool) string { if !validID { return "quarantine" }; if seen { return "duplicate" }; return "apply" }
