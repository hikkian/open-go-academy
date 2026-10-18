package learning

import (
	"strings"
	"testing"

	"academy/internal/content"
	"academy/internal/store"
)

func TestCommunicationRequiresOpaqueSeniorReviewEvidence(t *testing.T) {
	catalog, err := content.Load()
	if err != nil {
		t.Fatal(err)
	}
	base := Build(store.Progress{Completed: []string{"p5:automatic"}, Evidence: map[string]int64{"p5:automatic": 17}})
	base = WithCatalog(base, store.Progress{Completed: []string{"p5:automatic"}, Evidence: map[string]int64{"p5:automatic": 17}}, catalog)
	assertCommunication := func(snapshot Snapshot, wantLevel int, wantEvidence int) {
		t.Helper()
		for _, skill := range snapshot.Skills {
			if skill.ID == "communication" {
				if skill.Level != wantLevel || len(skill.Evidence) != wantEvidence {
					t.Fatalf("communication evidence: level=%d evidence=%v; want level=%d count=%d", skill.Level, skill.Evidence, wantLevel, wantEvidence)
				}
				if snapshot.Verified || snapshot.CurrentRank != "Zero" {
					t.Fatalf("skill evidence awarded a rank: %+v", snapshot)
				}
				return
			}
		}
		t.Fatal("communication skill missing")
	}
	assertCommunication(base, 0, 0)

	invalid := []string{"", "approved", "sha256:bad", "sha256:" + strings.Repeat("g", 64)}
	for _, id := range invalid {
		assertCommunication(WithSeniorCommunicationEvidence(base, id), 0, 0)
	}

	proofID := "sha256:" + strings.Repeat("a", 64)
	withProof := WithSeniorCommunicationEvidence(base, proofID)
	assertCommunication(withProof, 4, 1)
	assertCommunication(WithSeniorCommunicationEvidence(withProof, proofID), 4, 1)
}
