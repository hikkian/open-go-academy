package main

func RedisQueueReviewHard(persistenceConfigured, writeAcknowledged, consumerRecovery, idempotentHandler, lossBudgetDefined bool) string {
	if persistenceConfigured && writeAcknowledged && consumerRecovery && idempotentHandler && lossBudgetDefined { return "candidate_only" }; return "needs_review"
}
