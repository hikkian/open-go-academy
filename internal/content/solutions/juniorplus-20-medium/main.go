package main

func EvidenceStage(automaticPassed,cleanSetup,testsAndDocs bool) string { if !automaticPassed || !cleanSetup || !testsAndDocs{return "incomplete"};return "manual_review_required" }
