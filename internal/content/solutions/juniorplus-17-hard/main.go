package main

func ReadinessState(databaseConnected, migrationsCurrent bool) string { if !databaseConnected { return "not_ready" }; if !migrationsCurrent { return "migrations_pending" }; return "ready" }
