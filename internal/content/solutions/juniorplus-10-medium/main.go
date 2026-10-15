package main

func LinkSortColumn(name string) string { switch name { case "slug": return "slug"; case "created": return "created_at"; case "status": return "enabled"; default: return "created_at" } }
