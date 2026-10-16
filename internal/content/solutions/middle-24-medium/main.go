package main

func RedisQueuePrerequisitesMedium(persistenceConfigured, writeAcknowledged, consumerRecovery, idempotentHandler bool) string {
	if persistenceConfigured && writeAcknowledged && consumerRecovery && idempotentHandler { return "candidate_only" }; return "needs_review"
}
