package main

func TaskAccess(authenticated, sameTenant, isOwner, canRead bool) string {
	if !authenticated {
		return "unauthenticated"
	}
	if !sameTenant {
		return "not_found"
	}
	if !isOwner && !canRead {
		return "forbidden"
	}
	return "allow"
}
