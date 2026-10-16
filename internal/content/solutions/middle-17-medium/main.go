package main

func OutboxActionMedium(committed bool, brokerAccepted bool, rowMarked bool) string {
	if !committed { return "discard" }
	if !brokerAccepted { return "publish" }
	if !rowMarked { return "mark_published" }
	return "complete"
}
