package main

func OutboxActionEasy(committed bool, published bool) string {
	if !committed { return "discard" }
	if !published { return "publish" }
	return "complete"
}
