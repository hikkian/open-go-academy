package main

func Revoke(exposed, impactHigh bool) string { if exposed && impactHigh { return "revoke-now" }; if exposed { return "rotate-and-review" }; return "monitor" }
