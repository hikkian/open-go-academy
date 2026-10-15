package main

func Authorize(role string, auth bool, principalID, ownerID int64, action string) string {
	if !auth {
		return "unauthenticated"
	}
	if principalID <= 0 || ownerID <= 0 {
		return "forbidden"
	}
	if action != "read" && action != "update" && action != "delete" {
		return "forbidden"
	}
	if role == "admin" {
		return "allow"
	}
	if role != "user" || principalID != ownerID {
		return "forbidden"
	}
	return "allow"
}
