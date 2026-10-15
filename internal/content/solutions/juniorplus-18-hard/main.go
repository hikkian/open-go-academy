package main

func ComposeStartupState(dbHealthy, migrationsApplied, apiHealthy bool) string { if !dbHealthy { return "wait_database" }; if !migrationsApplied { return "migration_failed" }; if !apiHealthy { return "api_unhealthy" }; return "ready" }
