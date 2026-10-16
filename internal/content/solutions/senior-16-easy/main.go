package main

func ScaleOption(stateful, coordinationAvailable bool) string {
	if !stateful {
		return "replicas_candidate"
	}
	if coordinationAvailable {
		return "partition_candidate"
	}
	return "state_boundary_first"
}
