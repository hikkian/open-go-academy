package main

func ReadAfterWrite(wrote, replicaFresh bool) string {
	if wrote || !replicaFresh {
		return "primary"
	}
	return "replica"
}
