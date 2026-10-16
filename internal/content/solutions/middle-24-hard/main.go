package main

func RedisQueueReviewHard(persistenceConfigured, producerAckDefined, ownershipRecoveryDefined, idempotentHandler, failoverTestDefined, recoveryTargetDefined, lossBudgetDefined bool) string {
	if persistenceConfigured && producerAckDefined && ownershipRecoveryDefined && idempotentHandler && failoverTestDefined && recoveryTargetDefined && lossBudgetDefined {
		return "candidate_only"
	}
	return "needs_review"
}
