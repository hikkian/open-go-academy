package main

func RetentionPolicy(classification, required string) string {
	if classification == "" || required == "" {
		return "clarify"
	}
	if classification == "regulated" && required == "durable" {
		return "review_retention"
	}
	if required == "ephemeral" {
		return "expire"
	}
	return "retain_by_policy"
}
