package main

func OperationState(committed, published bool) string { if !committed { return "pending" }; if !published { return "committed-unpublished" }; return "published" }
