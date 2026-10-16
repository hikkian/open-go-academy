package main

func LayerForOperationEasy(layer, operation string) string {
	if layer == "handler" && (operation == "decode" || operation == "encode") { return "allow" }; if layer == "service" && operation == "domain_rule" { return "allow" }; if layer == "repository" && operation == "sql_write" { return "allow" }; return "misplaced"
}
