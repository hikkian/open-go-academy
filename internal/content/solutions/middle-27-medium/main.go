package main

func OwnerDecision(principalID, ownerID string, authenticated bool) string {
	if !authenticated {
		return "unauthenticated"
	}
	if principalID == "" || ownerID == "" {
		return "invalid"
	}
	if principalID != ownerID {
		return "forbidden"
	}
	return "allow"
}
