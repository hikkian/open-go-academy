package main

func ConsistencySource(wrote, needsLatest, replicaFresh bool) string {
	if wrote || needsLatest || !replicaFresh {
		return "primary"
	}
	return "replica"
}
