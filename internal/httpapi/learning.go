package httpapi

import (
	"academy/internal/learning"
	"academy/internal/store"
)

func (a *API) learningSnapshot(progress store.Progress) (learning.Snapshot, []store.RankAward, error) {
	awards, err := a.Store.RankAwards("local")
	if err != nil {
		return learning.Snapshot{}, nil, err
	}
	reviewEvidenceID, err := a.Store.VerifiedSeniorCommunicationEvidence("local")
	if err != nil {
		return learning.Snapshot{}, nil, err
	}
	snapshot := learning.WithCatalog(learning.Build(progress), progress, a.Content)
	snapshot = learning.WithSeniorCommunicationEvidence(snapshot, reviewEvidenceID)
	return learning.WithAwards(snapshot, awards), awards, nil
}
