package main

func TagAssociationOutcome(linkOwner,tagOwner string,committed bool) string { if linkOwner=="" || tagOwner=="" || linkOwner!=tagOwner{return "rejected"};if !committed{return "rolled_back"};return "linked" }
