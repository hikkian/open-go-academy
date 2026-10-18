package learning

import (
	"encoding/hex"
	"strings"
)

// WithSeniorCommunicationEvidence adds communication practice only when the
// caller supplies an opaque proof produced from a persisted Senior award's
// exact P5 human-review snapshot. Automatic project results never call this.
func WithSeniorCommunicationEvidence(snapshot Snapshot, evidenceID string) Snapshot {
	if !isEvidenceDigest(evidenceID) {
		return snapshot
	}
	for i := range snapshot.Skills {
		skill := &snapshot.Skills[i]
		if skill.ID != "communication" {
			continue
		}
		found := false
		for _, existing := range skill.Evidence {
			found = found || existing == evidenceID
		}
		if !found {
			skill.Evidence = append(skill.Evidence, evidenceID)
		}
		if skill.Level < 4 {
			skill.Level = 4
		}
	}
	return snapshot
}

func isEvidenceDigest(value string) bool {
	if !strings.HasPrefix(value, "sha256:") {
		return false
	}
	digest := strings.TrimPrefix(value, "sha256:")
	if len(digest) != 64 {
		return false
	}
	_, err := hex.DecodeString(digest)
	return err == nil
}
