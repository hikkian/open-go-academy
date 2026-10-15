package main

func ConstraintCategory(sqlState string) string { switch sqlState { case "23505": return "unique_conflict"; case "23503": return "missing_reference"; case "23514": return "check_violation"; default: return "storage_error" } }
