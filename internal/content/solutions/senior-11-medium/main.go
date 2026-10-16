package main

func RateScope(authenticated, tenantKnown bool) string { if !authenticated { return "anonymous" }; if !tenantKnown { return "invalid-scope" }; return "tenant" }
