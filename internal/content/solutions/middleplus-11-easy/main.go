package main

func ReadPolicy(wrote, replicaFresh bool) string {
	if wrote || !replicaFresh {
		return "primary"
	}
	return "replica"
}
