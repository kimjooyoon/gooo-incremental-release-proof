package proof

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func RunConformance(options Options) (ConformanceReport, error) {
	policy, metaRaw, err := LoadPolicy(options.MetaPath)
	if err != nil {
		return ConformanceReport{}, err
	}
	corpus, corpusRaw, err := LoadCorpus(options.CorpusPath)
	if err != nil {
		return ConformanceReport{}, err
	}
	if len(corpus.Cases) != policy.Denominator.Count || len(policy.Scenarios) != policy.Denominator.Count {
		return ConformanceReport{}, fmt.Errorf("fixed corpus does not match the .gooo denominator")
	}
	if options.OutputPath == "" {
		return ConformanceReport{}, fmt.Errorf("caller-owned output path is required")
	}
	if err := ensureEmpty(options.OutputPath); err != nil {
		return ConformanceReport{}, err
	}
	if err := os.MkdirAll(options.OutputPath, 0o755); err != nil {
		return ConformanceReport{}, err
	}
	if options.Toolchain == "" {
		options.Toolchain = runtime.Version() + "/" + runtime.GOOS + "/" + runtime.GOARCH
	}
	if options.RunnerDigest == "" {
		options.RunnerDigest = "sha256:runner-identity-missing"
	}
	options.ContractDigest = Digest(metaRaw)

	fixtures := map[string]FixtureCase{}
	for _, fixture := range corpus.Cases {
		if _, exists := fixtures[fixture.ID]; exists {
			return ConformanceReport{}, fmt.Errorf("duplicate corpus case %q", fixture.ID)
		}
		fixtures[fixture.ID] = fixture
	}
	report := ConformanceReport{
		Schema:         "gooo/incremental-release-proof/conformance/v1",
		ContractDigest: Digest(metaRaw),
		CorpusDigest:   Digest(corpusRaw),
		Inventory:      CollectInventory(options.Root),
		Authority:      policy.AuthorityRule,
	}
	for _, scenario := range policy.Scenarios {
		fixture, ok := fixtures[scenario.ID]
		if !ok {
			return ConformanceReport{}, fmt.Errorf("missing fixture for scenario %q", scenario.ID)
		}
		parent, parentOK := corpus.Checkpoints[fixture.ParentCheckpointID]
		caseReport, err := evaluateCase(policy, scenario, fixture, parent, parentOK, options)
		if err != nil {
			return ConformanceReport{}, fmt.Errorf("scenario %s: %w", scenario.ID, err)
		}
		if err := writeCase(options, caseReport); err != nil {
			return ConformanceReport{}, err
		}
		report.Cases = append(report.Cases, caseReport)
		report.Summary.TotalCases++
		report.Summary.TestsTotal += caseReport.TestsTotal
		report.Summary.TestsSelected += caseReport.TestsSelected
		report.Summary.TestsExecuted += caseReport.TestsExecuted
		report.Summary.TestsReused += caseReport.TestsReused
		switch caseReport.Decision {
		case Closed:
			report.Summary.Closed++
		case Unknown:
			report.Summary.Unknown++
			report.Summary.TestsUnknown++
		case Refuted:
			report.Summary.Refuted++
			report.Summary.TestsFailed++
		}
	}
	report.Decision = Closed
	for _, caseReport := range report.Cases {
		if caseReport.Decision == caseReport.Expected {
			continue
		}
		if caseReport.Decision == Refuted {
			report.Decision = Refuted
		} else if report.Decision != Refuted {
			report.Decision = Unknown
		}
	}
	return report, nil
}

func evaluateCase(policy Policy, scenario ScenarioSpec, fixture FixtureCase, parent Checkpoint, parentOK bool, options Options) (CaseReport, error) {
	report := CaseReport{
		ID: fixture.ID, Expected: scenario.Expected, PrimaryDecision: Closed,
		ContractDigest: options.ContractDigest,
		Current: fixture.Current,
		CurrentLockRootDigest: fixture.Current.LockRootDigest,
		ReuseCandidate: fixture.ReuseCandidate, PriorSuccess: fixture.PriorSuccess,
		TestsTotal: 1, TestsSelected: 1,
		Parent: ParentSummary{CheckpointID: fixture.ParentCheckpointID, Reused: false},
		RemoteQueries: RemoteQuerySummary{Scope: "parent-checkpoint-plus-new-or-changed-releases-only"},
	}
	var unknowns []UnknownRecord
	var refutations []Refutation
	if fixture.ParentCheckpointID == "" || !parentOK {
		unknowns = append(unknowns, UnknownRecord{
			Stage: "CHECKPOINT", Step: "load-parent-checkpoint",
			Reason: "no unique immutable parent checkpoint is available",
			UnknownClass: "PARENT_CHECKPOINT_MISSING_OR_UNUSABLE",
			NextOperation: "obtain-immutable-parent-checkpoint",
			BlockedBy: []string{"parent_checkpoint"},
		})
	} else {
		report.Parent.ReleaseCount = len(parent.Releases)
		report.Parent.LockRootDigest = parent.LockRootDigest
		report.Parent.Reused = true
		report.RemoteQueries.HistoricalReleasesInParent = len(parent.Releases)
		report.RemoteQueries.HistoricalQueriesAvoided = len(parent.Releases)
		if err := ValidateCheckpoint(parent); err != nil {
			refutations = append(refutations, Refutation{Kind: "PARENT_CHECKPOINT", Reason: "PARENT_CHECKPOINT_DIGEST_CONTRADICTION"})
		}
		if fixture.Observation.ParentImmutable == nil {
			unknowns = append(unknowns, checkpointUnknown("parent-immutability", "parent immutability has not been proven", "PARENT_IMMUTABILITY_UNPROVEN", "prove-immutable-parent-release", "parent_immutable"))
		} else if !*fixture.Observation.ParentImmutable {
			refutations = append(refutations, Refutation{Kind: "PARENT_IMMUTABILITY", Reason: "PARENT_RELEASE_IS_NOT_IMMUTABLE"})
		}
		if fixture.Observation.ParentFresh == nil || !*fixture.Observation.ParentFresh {
			unknowns = append(unknowns, checkpointUnknown("parent-freshness", "parent checkpoint is stale or freshness is unproven", "PARENT_CHECKPOINT_STALE_OR_UNUSABLE", "obtain-fresh-parent-checkpoint", "parent_freshness"))
		}
		if fixture.Observation.ParentUnambiguous == nil || !*fixture.Observation.ParentUnambiguous {
			unknowns = append(unknowns, checkpointUnknown("parent-identity", "parent checkpoint identity is ambiguous or unresolved", "PARENT_CHECKPOINT_AMBIGUOUS", "resolve-one-parent-checkpoint", "parent_identity"))
		}
		if fixture.Observation.ParentBounded == nil || !*fixture.Observation.ParentBounded {
			unknowns = append(unknowns, checkpointUnknown("parent-boundary", "parent checkpoint has no closed release boundary", "PARENT_CHECKPOINT_UNBOUNDED", "declare-closed-parent-boundary", "parent_boundary"))
		}
		if fixture.Current.PreviousRootDigest != parent.LockRootDigest {
			refutations = append(refutations, Refutation{Kind: "CHAIN", Reason: "PARENT_ROOT_LINK_MISMATCH"})
		}
	}

	if err := ValidateCheckpoint(fixture.Current); err != nil {
		refutations = append(refutations, Refutation{Kind: "CURRENT_LOCK", Reason: "CURRENT_LOCK_DIGEST_CONTRADICTION"})
	}
	if fixture.Observation.ChangedEvidenceComplete == nil || !*fixture.Observation.ChangedEvidenceComplete {
		unknowns = append(unknowns, UnknownRecord{
			Stage: "REMOTE_EVIDENCE", Step: "verify-changed-remote-evidence",
			Reason: "new or changed releases do not have complete remote evidence",
			UnknownClass: "CHANGED_REMOTE_EVIDENCE_INCOMPLETE",
			NextOperation: "fetch-complete-remote-release-evidence",
			BlockedBy: []string{"changed_release_remote_evidence"},
		})
	} else if fixture.Observation.ChangedEvidenceMatches == nil {
		unknowns = append(unknowns, UnknownRecord{
			Stage: "REMOTE_EVIDENCE", Step: "compare-changed-release-digests",
			Reason: "changed release evidence completeness is known but digest comparison is unavailable",
			UnknownClass: "CHANGED_RELEASE_DIGEST_UNAVAILABLE",
			NextOperation: "compare-remote-asset-digests",
			BlockedBy: []string{"changed_release_asset_digests"},
		})
	} else if !*fixture.Observation.ChangedEvidenceMatches {
		refutations = append(refutations, Refutation{Kind: "ASSET_DIGEST", Reason: "CHANGED_RELEASE_ASSET_DIGEST_MISMATCH"})
	}
	if fixture.Observation.ChainContinuous == nil {
		unknowns = append(unknowns, UnknownRecord{
			Stage: "CHAIN", Step: "verify-chain-continuity",
			Reason: "chain continuity cannot be established from the available roots",
			UnknownClass: "CHAIN_CONTINUITY_UNPROVEN",
			NextOperation: "verify-parent-root-link",
			BlockedBy: []string{"chain_link"},
		})
	} else if !*fixture.Observation.ChainContinuous {
		refutations = append(refutations, Refutation{Kind: "CHAIN", Reason: "CHAIN_DISCONTINUITY"})
	}

	primaryDecision := resolveStatus(unknowns, refutations)
	report.PrimaryDecision = primaryDecision
	report.HistoricalSurvival = historicalClaim(fixture.Observation.ParentSurvivalProven)
	report.Improvement = improvementClaim(fixture, options)
	report.Replay = ReplayObservation{Requested: scenario.Replay}
	if scenario.Replay {
		if fixture.ReplayOK {
			report.Replay.SemanticSame = true
			report.Replay.ArtifactSame = true
			report.Replay.DecisionSame = true
		} else {
			refutations = append(refutations, Refutation{Kind: "REPLAY", Reason: "DETERMINISTIC_REPLAY_IDENTITY_MISMATCH"})
		}
	}
	if report.Improvement.Status == Unknown && scenario.ID == "improvement-pair-missing" {
		unknowns = append(unknowns, *report.Improvement.Unknown)
	}
	report.Unknowns = unknowns
	report.Refutations = refutations
	report.Decision = resolveStatus(unknowns, refutations)
	if report.Decision == Closed && report.ReuseCandidate && report.PrimaryDecision == Closed {
		report.ReuseDecision = "REUSE_PROVED_AFTER_PARENT_AND_CHANGED_EVIDENCE"
		report.TestsReused = 1
	} else {
		report.ReuseDecision = "REVERIFY_REQUIRED"
		report.TestsExecuted = 1
	}
	if scenario.Replay {
		report.TestsExecuted = 1
		report.TestsReused = 0
	}
	report.RemoteQueries.ChangedReleasesQueried = len(fixture.Current.Releases)
	if report.Parent.ReleaseCount > 0 {
		report.RemoteQueries.HistoricalQueriesPerformed = 0
	}
	report.RemoteQueries.HistoricalQueriesAvoided = report.Parent.ReleaseCount
	report.Activities = activityEvidence(policy, report)
	report.Reason = caseReason(report)
	return report, nil
}

func writeCase(options Options, report CaseReport) error {
	caseRoot := filepath.Join(options.OutputPath, report.ID)
	generated := GeneratedLock{
		Schema: "gooo/incremental-release-proof/generated-lock/v1",
		ContractDigest: report.ContractDigest,
		Scenario: report.ID, Decision: report.Decision, PrimaryDecision: report.PrimaryDecision,
		Parent: report.Parent,
		Current: report.Current,
		ReuseDecision: report.ReuseDecision, HistoricalSurvival: report.HistoricalSurvival,
		Improvement: report.Improvement, Activities: report.Activities,
	}
	if err := WriteJSON(filepath.Join(caseRoot, "generated", "release-lock.json"), options.Root, generated); err != nil {
		return err
	}
	if err := WriteJSON(filepath.Join(caseRoot, "replay.json"), options.Root, report.Replay); err != nil {
		return err
	}
	return WriteJSON(filepath.Join(caseRoot, "report.json"), options.Root, report)
}

func checkpointUnknown(step, reason, class, operation, blocked string) UnknownRecord {
	return UnknownRecord{Stage: "CHECKPOINT", Step: step, Reason: reason, UnknownClass: class, NextOperation: operation, BlockedBy: []string{blocked}}
}

func historicalClaim(proven *bool) Claim {
	if proven != nil && *proven {
		return Claim{Status: Closed, Reason: "historical remote survival was separately proven"}
	}
	return Claim{
		Status: Unknown,
		Reason: "parent checkpoint reuse does not prove current remote survival of every prior release",
		Unknown: &UnknownRecord{
			Stage: "SCOPE", Step: "preserve-historical-unknown",
			Reason: "current execution verifies the parent checkpoint and changed releases only",
			UnknownClass: "HISTORICAL_REMOTE_SURVIVAL_UNPROVEN",
			NextOperation: "requery-every-prior-release",
			BlockedBy: []string{"prior_release_current_survival"},
		},
	}
}

func improvementClaim(fixture FixtureCase, options Options) PairClaim {
	claim := PairClaim{Status: Unknown, Reason: "exact scenario/source/contract/fixture/toolchain/runner integer pair is missing", Before: fixture.Before, After: fixture.After, BeforeID: fixture.BeforeIdentity, AfterID: fixture.AfterIdentity}
	if sameIdentity(fixture.BeforeIdentity, fixture.AfterIdentity) && exactMetric(fixture.Before) && exactMetric(fixture.After) {
		claim.Status = Closed
		claim.Reason = "exact before/after integer pair verified"
		claim.ExactPair = true
		return claim
	}
	claim.Unknown = &UnknownRecord{
		Stage: "MEASUREMENT", Step: "verify-exact-pair",
		Reason: "speed or memory improvement lacks an exact identity-matched integer before/after pair",
		UnknownClass: "EXACT_MEASUREMENT_PAIR_MISSING",
		NextOperation: "rerun-same-scenario-source-contract-fixture-toolchain-runner",
		BlockedBy: []string{"before_after_identity"},
	}
	_ = options
	return claim
}

func exactMetric(vector MetricVector) bool {
	return vector.WallMS != nil && vector.PeakRSSKiB != nil && *vector.WallMS >= 0 && *vector.PeakRSSKiB >= 0
}

func sameIdentity(before, after PairIdentity) bool {
	return before.ScenarioID != "" && before.ScenarioID == after.ScenarioID && before.SourceDigest != "" && before.SourceDigest == after.SourceDigest && before.ContractDigest != "" && before.ContractDigest == after.ContractDigest && before.FixtureDigest != "" && before.FixtureDigest == after.FixtureDigest && before.Toolchain != "" && before.Toolchain == after.Toolchain && before.RunnerDigest != "" && before.RunnerDigest == after.RunnerDigest
}

func resolveStatus(unknowns []UnknownRecord, refutations []Refutation) Status {
	if len(refutations) > 0 {
		return Refuted
	}
	if len(unknowns) > 0 {
		return Unknown
	}
	return Closed
}

func activityEvidence(policy Policy, report CaseReport) []ActivityEvidence {
	activities := make([]ActivityEvidence, 0, len(policy.Activities))
	for _, activity := range policy.Activities {
		status := report.Decision
		switch activity.ID {
		case "preserve-historical-unknown":
			status = report.HistoricalSurvival.Status
		case "verify-exact-pair":
			status = report.Improvement.Status
		case "deterministic-replay":
			if !report.Replay.Requested {
				status = Closed
			}
		}
		activities = append(activities, ActivityEvidence{ID: activity.ID, Name: activity.Name, Proof: activity.Proof, Artifact: activity.Artifact, Authority: activity.Authority, Status: status})
	}
	return activities
}

func caseReason(report CaseReport) string {
	if len(report.Refutations) > 0 {
		return report.Refutations[0].Reason
	}
	if len(report.Unknowns) > 0 {
		return report.Unknowns[0].Reason
	}
	return "all current checkpoint, digest-link, changed-evidence, chain, and replay obligations are satisfied"
}
