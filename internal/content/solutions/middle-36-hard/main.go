package main

func SameProject(taskProject, requestedProject string) bool {
	return taskProject != "" && requestedProject != "" && taskProject == requestedProject
}
