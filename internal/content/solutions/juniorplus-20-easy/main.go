package main

func FailureCategory(check string) string { switch check { case "http_contract":return "http";case "auth_session":return "authentication";case "owner_scope":return "ownership";case "postgres_integrity":return "database";case "redirect_boundary":return "redirect";default:return "unknown" } }
