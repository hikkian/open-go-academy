package main

func DependencyPlan(repositoryProvided, serviceProvided, handlerProvided bool) string {
	if !repositoryProvided {
		return "missing_repository"
	}
	if !serviceProvided {
		return "missing_service"
	}
	if !handlerProvided {
		return "missing_handler"
	}
	return "ready"
}
