package main

func SameTenantScope(principalTenant, resourceTenant string) bool { return principalTenant != "" && resourceTenant != "" && principalTenant == resourceTenant }
