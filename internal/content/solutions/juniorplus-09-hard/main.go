package main

func LinkColumnAllowed(name string) bool { switch name { case "owner_id", "slug", "target_url", "enabled", "created_at": return true; default: return false } }
