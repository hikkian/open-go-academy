package main

func CanAccessResource(ownerID, actorID int, role string) bool {
	if ownerID <= 0 || actorID <= 0 {
		return false
	}
	return role == "admin" || (role == "user" && actorID == ownerID)
}
