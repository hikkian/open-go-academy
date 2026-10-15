package main

func CreateLinkStatement() string { return "INSERT INTO links(owner_id, slug, target_url) VALUES ($1, $2, $3) RETURNING id" }
