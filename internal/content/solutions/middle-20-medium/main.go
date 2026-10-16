package main

func DirectRouteMedium(exchange, routingKey, bindingKey string) string {
	if routingKey == "" || bindingKey == "" { return "invalid" }; if exchange == "" { return "invalid" }; if routingKey == bindingKey { return "route" }; return "unrouted"
}
