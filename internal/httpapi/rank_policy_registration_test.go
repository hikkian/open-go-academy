package httpapi

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"academy/internal/assessment"
	"academy/internal/projectreview"
	"academy/internal/projectrunner"
	"academy/internal/projects"
	"academy/internal/runner"
	"academy/internal/store"
)

type rankPolicyProfileRunner struct {
	profiles map[string]projectrunner.EvaluationProfile
}

func (r *rankPolicyProfileRunner) Available(context.Context) bool { return true }
func (r *rankPolicyProfileRunner) Execute(context.Context, projectrunner.Request) (runner.Result, error) {
	return runner.Result{}, nil
}
func (r *rankPolicyProfileRunner) ProjectProfile(_ context.Context, projectID string) (projectrunner.EvaluationProfile, bool) {
	profile, ok := r.profiles[projectID]
	return profile, ok
}

func newRankPolicyRegistry(t *testing.T, reviewerIDs ...string) *projectreview.Registry {
	t.Helper()
	entries := make([]map[string]string, 0, len(reviewerIDs))
	for _, id := range reviewerIDs {
		public, _, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, map[string]string{"reviewer_id": id, "public_key": base64.StdEncoding.EncodeToString(public)})
	}
	body, err := json.Marshal(map[string]any{"reviewers": entries})
	if err != nil {
		t.Fatal(err)
	}
	registry, err := projectreview.LoadRegistry(strings.NewReader(string(body)))
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func newRankPolicyTestAPI(t *testing.T, reviewers *projectreview.Registry, profiles map[string]projectrunner.EvaluationProfile) *API {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	for _, exam := range assessment.All() {
		if err := s.RegisterExam(exam.ID, exam.Version, exam.Rank, exam.Hash()); err != nil {
			t.Fatal(err)
		}
	}
	return &API{Store: s, ProjectReviewers: reviewers, ProjectRunner: &rankPolicyProfileRunner{profiles: profiles}}
}

func rankPolicyTestProfiles() map[string]projectrunner.EvaluationProfile {
	profiles := map[string]projectrunner.EvaluationProfile{}
	for _, id := range []string{"p3", "p4", "p5"} {
		project, _ := projects.Find(id)
		criteria, _ := projects.CriteriaHash(id)
		profiles[id] = projectrunner.EvaluationProfile{
			ProjectVersion: project.Version, CriteriaHash: criteria,
			CheckerVersion: id + "-checker-v1", CheckerDigest: "sha256:" + strings.Repeat("a", 64),
			RuntimePolicyHash: "sha256:" + strings.Repeat("b", 64),
			RubricVersion:     id + "-automatic-v1", RubricHash: "sha256:" + strings.Repeat("c", 64),
		}
	}
	return profiles
}

func TestAdvancedRankPolicyRegistrationFailsClosedWithoutRegistryOrProfile(t *testing.T) {
	for _, test := range []struct {
		name          string
		registry      *projectreview.Registry
		profiles      map[string]projectrunner.EvaluationProfile
		wantAvailable []string
	}{
		{name: "missing registry", profiles: rankPolicyTestProfiles()},
		{name: "missing project profile", registry: newRankPolicyRegistry(t, "reviewer-a", "reviewer-b"), profiles: map[string]projectrunner.EvaluationProfile{}},
		{name: "profile bound to another project version", registry: newRankPolicyRegistry(t, "reviewer-a", "reviewer-b"), profiles: func() map[string]projectrunner.EvaluationProfile {
			profiles := rankPolicyTestProfiles()
			p := profiles["p3"]
			p.ProjectVersion = "unexpected-version"
			profiles["p3"] = p
			return profiles
		}(), wantAvailable: []string{"Middle+", "Senior"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			api := newRankPolicyTestAPI(t, test.registry, test.profiles)
			if err := api.registerAdvancedRankPolicies(context.Background()); err != nil {
				t.Fatalf("registration should safely skip unavailable evidence: %v", err)
			}
			available := map[string]bool{}
			for _, exam := range assessment.All() {
				if !advancedRank(exam.Rank) {
					continue
				}
				_, err := api.Store.RankPolicyForExam(exam.ID, exam.Version)
				if err == nil {
					available[exam.Rank] = true
				}
			}
			if len(available) != len(test.wantAvailable) {
				t.Fatalf("unexpected policy availability: got %v want %v", available, test.wantAvailable)
			}
			for _, rank := range test.wantAvailable {
				if !available[rank] {
					t.Fatalf("expected policy for %s; got %v", rank, available)
				}
			}
		})
	}
}

func TestAdvancedRankPoliciesPinExamProjectRubricAndTrust(t *testing.T) {
	registry := newRankPolicyRegistry(t, "reviewer-a", "reviewer-b")
	api := newRankPolicyTestAPI(t, registry, rankPolicyTestProfiles())
	if err := api.registerAdvancedRankPolicies(context.Background()); err != nil {
		t.Fatal(err)
	}
	want := map[string]struct {
		examID, projectID string
		quorum            int
	}{
		"Middle":  {"middle-exam", "p3", 1},
		"Middle+": {"middleplus-exam", "p4", 1},
		"Senior":  {"senior-exam", "p5", 2},
	}
	for rank, binding := range want {
		exam, _ := assessment.Find(binding.examID, "")
		policy, err := api.Store.RankPolicyForExam(exam.ID, exam.Version)
		if err != nil {
			t.Fatalf("policy %s: %v", rank, err)
		}
		if policy.Rank != rank || policy.CriteriaHash != exam.Hash() || !reflect.DeepEqual(policy.RequiredCourseExerciseIDs, exam.Prerequisites) || !reflect.DeepEqual(policy.RequiredPriorRanks, []string{exam.RequiredRank}) {
			t.Fatalf("policy %s did not pin its exact exam inputs: %+v", rank, policy)
		}
		if len(policy.ProjectEvidenceRequirements) != 1 {
			t.Fatalf("policy %s requirements = %+v", rank, policy.ProjectEvidenceRequirements)
		}
		requirement := policy.ProjectEvidenceRequirements[0]
		project, _ := projects.Find(binding.projectID)
		criteria, _ := projects.CriteriaHash(binding.projectID)
		profile := rankPolicyTestProfiles()[binding.projectID]
		rubric := buildManualReviewRubric(project)
		fingerprints := registry.KeyFingerprints()
		if requirement.ProjectID != binding.projectID || requirement.ProjectVersion != project.Version || requirement.CriteriaHash != criteria || requirement.CheckerVersion != profile.CheckerVersion || requirement.CheckerDigest != profile.CheckerDigest || requirement.RuntimePolicyHash != profile.RuntimePolicyHash || requirement.RubricVersion != profile.RubricVersion || requirement.RubricHash != profile.RubricHash || !requirement.RequirePass {
			t.Fatalf("policy %s is not bound to the exact evaluator profile: %+v", rank, requirement)
		}
		if requirement.ManualReviewRubricVersion != rubric.Version || requirement.ManualReviewRubricHash != rubric.Hash || requirement.ReviewTrustPolicyVersion != "review-registry-v1" || requirement.ReviewTrustPolicyHash != store.ComputeReviewTrustPolicyHash("review-registry-v1", fingerprints) || !reflect.DeepEqual(requirement.AllowedReviewerKeyFingerprints, fingerprints) || requirement.MinimumDistinctReviewerKeyFingerprints != binding.quorum {
			t.Fatalf("policy %s has incomplete review/trust binding: %+v", rank, requirement)
		}
	}
	// Registration is idempotent and does not create extra immutable versions.
	if err := api.registerAdvancedRankPolicies(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, binding := range want {
		exam, _ := assessment.Find(binding.examID, "")
		policy, err := api.Store.RankPolicyForExam(exam.ID, exam.Version)
		if err != nil || policy.PolicyVersion != 1 {
			t.Fatalf("identical policy did not reuse its version: %+v, %v", policy, err)
		}
	}
}

func TestAdvancedRankPolicyTrustRotationCreatesNewVersion(t *testing.T) {
	api := newRankPolicyTestAPI(t, newRankPolicyRegistry(t, "reviewer-a", "reviewer-b"), rankPolicyTestProfiles())
	if err := api.registerAdvancedRankPolicies(context.Background()); err != nil {
		t.Fatal(err)
	}
	exam, _ := assessment.Find("middle-exam", "")
	old, err := api.Store.RankPolicyForExam(exam.ID, exam.Version)
	if err != nil {
		t.Fatal(err)
	}
	api.ProjectReviewers = newRankPolicyRegistry(t, "reviewer-c", "reviewer-d")
	if err := api.registerAdvancedRankPolicies(context.Background()); err != nil {
		t.Fatal(err)
	}
	latest, err := api.Store.RankPolicyForExam(exam.ID, exam.Version)
	if err != nil {
		t.Fatal(err)
	}
	previous, err := api.Store.RankPolicy(exam.ID, exam.Version, 1)
	if err != nil || previous.PolicyHash != old.PolicyHash {
		t.Fatalf("old policy version was changed: %+v, %v", previous, err)
	}
	if latest.PolicyVersion != 2 || latest.PolicyHash == old.PolicyHash || latest.ProjectEvidenceRequirements[0].ReviewTrustPolicyHash == old.ProjectEvidenceRequirements[0].ReviewTrustPolicyHash {
		t.Fatalf("reviewer rotation did not produce a new immutable policy: old=%+v latest=%+v", old, latest)
	}
}

func TestCurrentAndPinnedAttemptTrustFailClosedAfterRegistryChange(t *testing.T) {
	api := newRankPolicyTestAPI(t, newRankPolicyRegistry(t, "reviewer-a", "reviewer-b"), rankPolicyTestProfiles())
	if err := api.registerAdvancedRankPolicies(context.Background()); err != nil {
		t.Fatal(err)
	}
	exam, _ := assessment.Find("middle-exam", "")
	policy, err := api.Store.RankPolicyForExam(exam.ID, exam.Version)
	if err != nil {
		t.Fatal(err)
	}
	attempt := store.ExamAttempt{
		ExamID: exam.ID, Version: exam.Version,
		EvidenceJSON: fmt.Sprintf(`{"rank_policy_version":%d,"rank_policy_hash":%q}`, policy.PolicyVersion, policy.PolicyHash),
	}
	if !api.currentReviewTrustMatches(exam.ID, exam.Version) || !api.attemptReviewTrustMatches(attempt) {
		t.Fatal("matching current registry should permit policy and pinned attempt trust")
	}
	profileRunner := api.ProjectRunner.(*rankPolicyProfileRunner)
	profile := profileRunner.profiles["p3"]
	profile.CheckerDigest = "sha256:" + strings.Repeat("d", 64)
	profileRunner.profiles["p3"] = profile
	if api.currentReviewTrustMatches(exam.ID, exam.Version) || api.attemptReviewTrustMatches(attempt) {
		t.Fatal("changed evaluator profile must not trust stale policy or pending attempt")
	}
	profile.CheckerDigest = "sha256:" + strings.Repeat("a", 64)
	profileRunner.profiles["p3"] = profile
	api.ProjectReviewers = newRankPolicyRegistry(t, "reviewer-c", "reviewer-d")
	if api.currentReviewTrustMatches(exam.ID, exam.Version) || api.attemptReviewTrustMatches(attempt) {
		t.Fatal("rotated registry must not trust stale policy or pending attempt")
	}
	api.ProjectReviewers = nil
	if api.currentReviewTrustMatches(exam.ID, exam.Version) || api.attemptReviewTrustMatches(attempt) {
		t.Fatal("removed reviewer registry must fail closed for policy and pending attempt")
	}
}

func TestCurrentPolicyTrustFailsClosedWhenManualRubricChanges(t *testing.T) {
	api := newRankPolicyTestAPI(t, newRankPolicyRegistry(t, "reviewer-a", "reviewer-b"), rankPolicyTestProfiles())
	if err := api.registerAdvancedRankPolicies(context.Background()); err != nil {
		t.Fatal(err)
	}
	exam, _ := assessment.Find("middle-exam", "")
	policy, err := api.Store.RankPolicyForExam(exam.ID, exam.Version)
	if err != nil {
		t.Fatal(err)
	}
	if !api.currentReviewTrustMatches(exam.ID, exam.Version) {
		t.Fatal("initial exact manual rubric should match")
	}
	policy.PolicyVersion++
	policy.ProjectEvidenceRequirements[0].ManualReviewRubricHash = strings.Repeat("d", 64)
	if _, err := api.Store.RegisterRankPolicy(policy); err != nil {
		t.Fatal(err)
	}
	if api.currentReviewTrustMatches(exam.ID, exam.Version) {
		t.Fatal("current policy with changed manual rubric must fail closed")
	}
}

func TestPinnedAttemptTrustFailsClosedAfterSameTrustPolicyRotation(t *testing.T) {
	api := newRankPolicyTestAPI(t, newRankPolicyRegistry(t, "reviewer-a", "reviewer-b"), rankPolicyTestProfiles())
	if err := api.registerAdvancedRankPolicies(context.Background()); err != nil {
		t.Fatal(err)
	}
	exam, _ := assessment.Find("middle-exam", "")
	initial, err := api.Store.RankPolicyForExam(exam.ID, exam.Version)
	if err != nil {
		t.Fatal(err)
	}
	attempt := store.ExamAttempt{ExamID: exam.ID, Version: exam.Version,
		EvidenceJSON: fmt.Sprintf(`{"rank_policy_version":%d,"rank_policy_hash":%q}`, initial.PolicyVersion, initial.PolicyHash)}
	if !api.attemptReviewTrustMatches(attempt) {
		t.Fatal("current policy should accept its pinned attempt")
	}
	rotated := initial
	rotated.PolicyVersion++
	rotated.RequiredCourseExerciseIDs = append(append([]string(nil), initial.RequiredCourseExerciseIDs...), "middle-review-rotation-marker")
	if _, err := api.Store.RegisterRankPolicy(rotated); err != nil {
		t.Fatal(err)
	}
	if !api.currentReviewTrustMatches(exam.ID, exam.Version) {
		t.Fatal("same-key/profile policy rotation should leave current policy trust valid")
	}
	if api.attemptReviewTrustMatches(attempt) {
		t.Fatal("attempt pinned to old policy version must fail closed after rotation")
	}
}

func TestSeniorRankPolicyRequiresTwoConfiguredKeys(t *testing.T) {
	api := newRankPolicyTestAPI(t, newRankPolicyRegistry(t, "reviewer-a"), rankPolicyTestProfiles())
	if err := api.registerAdvancedRankPolicies(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, examID := range []string{"middle-exam", "middleplus-exam"} {
		exam, _ := assessment.Find(examID, "")
		if _, err := api.Store.RankPolicyForExam(exam.ID, exam.Version); err != nil {
			t.Fatalf("one reviewer should allow %s policy: %v", exam.Rank, err)
		}
	}
	senior, _ := assessment.Find("senior-exam", "")
	if _, err := api.Store.RankPolicyForExam(senior.ID, senior.Version); err == nil {
		t.Fatal("Senior policy was registered with fewer than two distinct keys")
	}
}

func TestAdvancedPolicyRegistrationSkipsProfileNotProvidedByRunner(t *testing.T) {
	profiles := rankPolicyTestProfiles()
	delete(profiles, "p4")
	api := newRankPolicyTestAPI(t, newRankPolicyRegistry(t, "reviewer-a", "reviewer-b"), profiles)
	if err := api.registerAdvancedRankPolicies(context.Background()); err != nil {
		t.Fatal(err)
	}
	middleplus, _ := assessment.Find("middleplus-exam", "")
	if _, err := api.Store.RankPolicyForExam(middleplus.ID, middleplus.Version); err == nil {
		t.Fatal("policy registered without exact P4 runner profile")
	}
	for _, examID := range []string{"middle-exam", "senior-exam"} {
		exam, _ := assessment.Find(examID, "")
		if _, err := api.Store.RankPolicyForExam(exam.ID, exam.Version); err != nil {
			t.Fatalf("unrelated profile absence blocked %s policy: %v", exam.Rank, err)
		}
	}
}

func Example_advancedRankPolicyProjectMapping() {
	fmt.Println(advancedRankProjects[0].rank + "=" + advancedRankProjects[0].projectID)
	// Output: Middle=p3
}
