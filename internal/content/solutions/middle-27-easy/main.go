package main

func AccessDecision(principalTenant, resourceTenant string) string {
	if principalTenant == "" || resourceTenant == "" {
		return "invalid"
	}
	if principalTenant != resourceTenant {
		return "deny"
	}
	return "tenant_match"
}
