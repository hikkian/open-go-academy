package main

func TopologyReadyEasy(exchange, queue, binding bool) bool {
	return exchange && queue && binding
}
