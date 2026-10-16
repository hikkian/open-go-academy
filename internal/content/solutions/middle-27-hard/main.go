package main

func AuthorizeTask(authenticated bool, role, operation, principalTenant, taskTenant, principalID, ownerID string) bool {
	if !authenticated {
		return false
	}
	if principalTenant == "" || taskTenant == "" || principalTenant != taskTenant {
		return false
	}
	if principalID == "" || ownerID == "" || principalID != ownerID {
		return false
	}
	if operation != "read" && operation != "write" {
		return false
	}
	switch role {
	case "owner", "editor":
		return true
	case "viewer":
		return operation == "read"
	default:
		return false
	}
}
