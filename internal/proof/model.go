package proof

type Status string

const (
	Closed  Status = "CLOSED"
	Unknown Status = "UNKNOWN"
	Refuted Status = "REFUTED"
)

type Policy struct {
	Schema        string
	Authority     string
	Precedence    []Status
	UnknownFields []string
	Denominator   Denominator
	Inventory     InventoryPolicy
	Metrics       MetricPolicy
	AuthorityRule AuthorityRule
	Generation    GenerationPlan
	Activities    []ActivitySpec
	Rules         map[string]RuleSpec
	Scenarios     []ScenarioSpec
}

type Denominator struct {
	ID    string
	Count int
}

type InventoryPolicy struct {
	RootReadme      string
	FileClasses     []string
	PhysicalLines   string
}

type MetricPolicy struct {
	Stages       []string
	Fields       []string
	TestCounts   []string
	Improvement  string
	Missing      string
}

type AuthorityRule struct {
	RepositoryWrites       int
	OutputScope            string
	AutomaticCommit        int
	AutomaticPush          int
	AutomaticMerge         int
	AutomaticRelease       int
	LocalVerification      string
}

type GenerationPlan struct {
	Language    string
	Role        string
	OutputScope string
}

type ActivitySpec struct {
	ID        string
	Name      string
	Proof     string
	Artifact  string
	Authority string
}

type RuleSpec struct {
	ID            string
	Status        Status
	Requires      string
	UnknownClass  string
	NextOperation string
	BlockedBy     string
}

type ScenarioSpec struct {
	ID     string
	Expected Status
	Rule   string
	Replay bool
}

type Asset struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	Digest string `json:"digest"`
}

type CIIdentity struct {
	RunID       int64  `json:"run_id"`
	RunSHA      string `json:"run_sha"`
	JobID       int64  `json:"job_id"`
	ArtifactID  int64  `json:"artifact_id"`
	ArtifactName string `json:"artifact_name"`
}

type ReleaseRecord struct {
	ImmutableReleaseID    int64       `json:"immutable_release_id"`
	Repository            string      `json:"repository"`
	Tag                   string      `json:"tag"`
	AnnotatedTagObjectSHA string      `json:"annotated_tag_object_sha"`
	PeeledCommitSHA       string      `json:"peeled_commit_sha"`
	CI                    CIIdentity  `json:"ci"`
	SourceAsset           Asset       `json:"source_asset"`
	EvidenceAsset         Asset       `json:"evidence_asset"`
}

type Checkpoint struct {
	Schema              string          `json:"schema"`
	CheckpointID        string          `json:"checkpoint_id"`
	PreviousRootDigest  string          `json:"previous_root_digest"`
	Releases            []ReleaseRecord `json:"releases"`
	MerkleRootDigest    string          `json:"merkle_root_digest"`
	LockRootDigest      string          `json:"lock_root_digest"`
}

type Observation struct {
	ParentImmutable         *bool `json:"parent_immutable"`
	ParentFresh             *bool `json:"parent_fresh"`
	ParentUnambiguous       *bool `json:"parent_unambiguous"`
	ParentBounded           *bool `json:"parent_bounded"`
	ChangedEvidenceComplete *bool `json:"changed_evidence_complete"`
	ChangedEvidenceMatches  *bool `json:"changed_evidence_matches"`
	ChainContinuous         *bool `json:"chain_continuous"`
	ParentSurvivalProven    *bool `json:"parent_survival_proven"`
}

type MetricVector struct {
	WallMS       *int64 `json:"wall_ms"`
	PeakRSSKiB   *int64 `json:"peak_rss_kib"`
}

type PairIdentity struct {
	ScenarioID       string `json:"scenario_id"`
	SourceDigest     string `json:"source_digest"`
	ContractDigest   string `json:"contract_digest"`
	FixtureDigest    string `json:"fixture_digest"`
	Toolchain        string `json:"toolchain"`
	RunnerDigest     string `json:"runner_digest"`
}

type PairClaim struct {
	Status      Status        `json:"status"`
	Reason      string        `json:"reason"`
	ExactPair   bool          `json:"exact_pair"`
	Before      MetricVector  `json:"before"`
	After       MetricVector  `json:"after"`
	BeforeID   PairIdentity   `json:"before_identity"`
	AfterID    PairIdentity   `json:"after_identity"`
	Unknown    *UnknownRecord `json:"unknown,omitempty"`
}

type Claim struct {
	Status  Status         `json:"status"`
	Reason  string         `json:"reason"`
	Unknown *UnknownRecord `json:"unknown,omitempty"`
}

type UnknownRecord struct {
	Stage         string   `json:"stage"`
	Step          string   `json:"step"`
	Reason        string   `json:"reason"`
	UnknownClass  string   `json:"unknown_class"`
	NextOperation string   `json:"next_operation"`
	BlockedBy     []string `json:"blocked_by"`
}

type Refutation struct {
	Kind   string `json:"kind"`
	Reason string `json:"reason"`
}

type ActivityEvidence struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Proof     string `json:"proof"`
	Artifact  string `json:"artifact"`
	Authority string `json:"authority"`
	Status    Status `json:"status"`
}

type RemoteQuerySummary struct {
	HistoricalReleasesInParent int `json:"historical_releases_in_parent"`
	HistoricalQueriesPerformed int `json:"historical_queries_performed"`
	ChangedReleasesQueried     int `json:"changed_releases_queried"`
	HistoricalQueriesAvoided   int `json:"historical_queries_avoided"`
	Scope                      string `json:"scope"`
}

type ParentSummary struct {
	CheckpointID   string `json:"checkpoint_id"`
	ReleaseCount   int    `json:"release_count"`
	LockRootDigest string `json:"lock_root_digest"`
	Reused         bool   `json:"reused"`
}

type ReplayObservation struct {
	Requested       bool   `json:"requested"`
	SemanticSame    bool   `json:"semantic_same"`
	ArtifactSame    bool   `json:"artifact_same"`
	DecisionSame    bool   `json:"decision_same"`
}

type FixtureCase struct {
	ID                 string        `json:"id"`
	ParentCheckpointID string        `json:"parent_checkpoint_id"`
	Current            Checkpoint    `json:"current"`
	Observation        Observation   `json:"observation"`
	ReuseCandidate     bool          `json:"reuse_candidate"`
	PriorSuccess       bool          `json:"prior_success"`
	Before             MetricVector  `json:"before"`
	After              MetricVector  `json:"after"`
	BeforeIdentity     PairIdentity  `json:"before_identity"`
	AfterIdentity      PairIdentity  `json:"after_identity"`
	ReplayOK           bool          `json:"replay_ok"`
}

type Corpus struct {
	Schema      string                 `json:"schema"`
	CorpusID    string                 `json:"corpus_id"`
	Checkpoints map[string]Checkpoint  `json:"checkpoints"`
	Cases       []FixtureCase          `json:"cases"`
}

type CaseReport struct {
	ID                    string             `json:"id"`
	Expected              Status             `json:"expected"`
	Decision              Status             `json:"decision"`
	PrimaryDecision       Status             `json:"primary_decision"`
	Reason                string             `json:"reason"`
	ContractDigest        string             `json:"contract_digest"`
	Parent                ParentSummary      `json:"parent"`
	Current               Checkpoint         `json:"current"`
	CurrentLockRootDigest string             `json:"current_lock_root_digest"`
	ReuseCandidate        bool               `json:"reuse_candidate"`
	PriorSuccess          bool               `json:"prior_success"`
	ReuseDecision         string             `json:"reuse_decision"`
	RemoteQueries         RemoteQuerySummary  `json:"remote_queries"`
	HistoricalSurvival    Claim              `json:"historical_remote_survival"`
	Improvement           PairClaim          `json:"improvement"`
	Replay                ReplayObservation  `json:"replay"`
	Unknowns              []UnknownRecord     `json:"unknowns"`
	Refutations           []Refutation        `json:"refutations"`
	Activities            []ActivityEvidence `json:"activities"`
	TestsTotal            int                `json:"tests_total"`
	TestsSelected         int                `json:"tests_selected"`
	TestsExecuted         int                `json:"tests_executed"`
	TestsReused           int                `json:"tests_reused"`
}

type ConformanceSummary struct {
	TotalCases    int `json:"total_cases"`
	Closed        int `json:"closed"`
	Unknown       int `json:"unknown"`
	Refuted       int `json:"refuted"`
	TestsTotal    int `json:"tests_total"`
	TestsSelected int `json:"tests_selected"`
	TestsExecuted int `json:"tests_executed"`
	TestsReused   int `json:"tests_reused"`
	TestsFailed   int `json:"tests_failed"`
	TestsUnknown  int `json:"tests_unknown"`
}

type Inventory struct {
	GoFiles           int64 `json:"go_files"`
	GoooFiles         int64 `json:"gooo_files"`
	GoPhysicalLines   int64 `json:"go_physical_lines"`
	GoooPhysicalLines int64 `json:"gooo_physical_lines"`
	DescendantDirs    int64 `json:"descendant_dirs"`
	RegularFiles      int64 `json:"regular_files"`
	RootReadmeExcluded bool `json:"root_readme_excluded"`
}

type ConformanceReport struct {
	Schema         string             `json:"schema"`
	ContractDigest string             `json:"contract_digest"`
	CorpusDigest   string             `json:"corpus_digest"`
	Decision       Status             `json:"decision"`
	Cases          []CaseReport       `json:"cases"`
	Summary        ConformanceSummary `json:"summary"`
	Inventory      Inventory          `json:"inventory"`
	Authority      AuthorityRule      `json:"authority"`
}

type GeneratedLock struct {
	Schema              string             `json:"schema"`
	ContractDigest      string             `json:"contract_digest"`
	Scenario            string             `json:"scenario"`
	Decision            Status             `json:"decision"`
	PrimaryDecision     Status             `json:"primary_decision"`
	Parent              ParentSummary      `json:"parent"`
	Current             Checkpoint         `json:"current"`
	ReuseDecision       string             `json:"reuse_decision"`
	HistoricalSurvival  Claim              `json:"historical_remote_survival"`
	Improvement         PairClaim          `json:"improvement"`
	Activities          []ActivityEvidence `json:"activities"`
}
