package main

func DesignReady(requirements, risks, rollback bool) bool {
	return requirements && risks && rollback
}
