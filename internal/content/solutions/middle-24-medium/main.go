package main

func RedisQueuePrerequisitesMedium(persistenceConfigured, producerAckDefined, ownershipRecoveryDefined, idempotentHandler bool) string {
	if persistenceConfigured && producerAckDefined && ownershipRecoveryDefined && idempotentHandler {
		return "candidate_only"
	}
	return "needs_review"
}
