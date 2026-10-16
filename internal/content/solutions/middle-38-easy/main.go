package main

func OutboxAction(committed, published bool) string {
	if !committed {
		return "not_committed"
	}
	if published {
		return "ack"
	}
	return "publish"
}
