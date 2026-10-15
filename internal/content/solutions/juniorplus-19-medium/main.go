package main

func OwnerAccess(authenticated bool,sessionOwner,resourceOwner string) string { if !authenticated{return "unauthorized"};if sessionOwner=="" || resourceOwner=="" || sessionOwner!=resourceOwner{return "not_found"};return "allow" }
