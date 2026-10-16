package main

func TaskInput(projectID, title string) string {
	if projectID == "" {
		return "missing_project"
	}
	if title == "" {
		return "missing_title"
	}
	if len(title) > 120 {
		return "title_too_long"
	}
	return "valid"
}
