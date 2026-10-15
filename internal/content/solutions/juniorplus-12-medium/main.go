package main

func RepositoryOutcome(sqlState string) string { switch sqlState { case "": return "success"; case "23505": return "conflict"; case "02000": return "not_found"; case "57014": return "canceled"; default: return "storage_error" } }
