// Package media defines provenance-preserving media ingestion contracts.
package media

import (
	"context"
	"errors"
	"time"
)

const (
	RunResultSchemaVersion = "nn.media.run-result/v1"
	ManifestSchemaVersion  = "nn.media.manifest/v1"
)

type RunID string
type BundleID string
type SourceID string
type ArtifactID string
type EvidenceID string

type Outcome string

const (
	OutcomeSucceeded          Outcome = "succeeded"
	OutcomePartial            Outcome = "partial"
	OutcomeInvocationFailed   Outcome = "invocation_failed"
	OutcomePrerequisiteFailed Outcome = "prerequisite_failed"
	OutcomeProcessingFailed   Outcome = "processing_failed"
	OutcomeCancelled          Outcome = "cancelled"
)

type StageState string

const (
	StageNotRequested StageState = "not_requested"
	StagePending      StageState = "pending"
	StageRunning      StageState = "running"
	StageSucceeded    StageState = "succeeded"
	StagePartial      StageState = "partial"
	StageInapplicable StageState = "inapplicable"
	StageUnavailable  StageState = "unavailable"
	StageFailed       StageState = "failed"
	StageInterrupted  StageState = "interrupted"
	StageCancelled    StageState = "cancelled"
)

type StageResult struct {
	Name      string     `json:"name,omitempty"`
	Requested bool       `json:"requested"`
	State     StageState `json:"state"`
}

type RunResult struct {
	SchemaVersion   string            `json:"schema_version"`
	Command         string            `json:"command,omitempty"`
	RunID           RunID             `json:"run_id,omitempty"`
	SourceID        SourceID          `json:"source_id,omitempty"`
	BundleID        BundleID          `json:"bundle_id,omitempty"`
	ManifestLocator string            `json:"manifest_locator,omitempty"`
	Outcome         Outcome           `json:"outcome"`
	Stages          []StageResult     `json:"stages,omitempty"`
	Warnings        []string          `json:"warnings,omitempty"`
	Errors          []StructuredError `json:"errors,omitempty"`
}

type StructuredError struct{ Code, Message, Stage string }

func DeriveOutcome(stages []StageResult) Outcome {
	hasSuccess, hasFailure := false, false
	for _, stage := range stages {
		if !stage.Requested {
			continue
		}
		switch stage.State {
		case StageCancelled, StageInterrupted:
			return OutcomeCancelled
		case StageUnavailable:
			if !hasSuccess {
				return OutcomePrerequisiteFailed
			}
			hasFailure = true
		case StageFailed:
			hasFailure = true
		case StageSucceeded, StageInapplicable:
			hasSuccess = true
		case StagePartial:
			hasSuccess, hasFailure = true, true
		}
	}
	if hasSuccess && hasFailure {
		return OutcomePartial
	}
	if hasFailure {
		return OutcomeProcessingFailed
	}
	return OutcomeSucceeded
}

func CanTransitionStage(from, to StageState) bool {
	allowed := map[StageState]map[StageState]bool{
		StageNotRequested: {StagePending: true},
		StagePending:      {StageRunning: true, StageCancelled: true, StageUnavailable: true, StageInapplicable: true},
		StageRunning:      {StageSucceeded: true, StagePartial: true, StageFailed: true, StageInterrupted: true, StageCancelled: true},
	}
	return from == to || allowed[from][to]
}

type LifecycleState string

const (
	LifecycleStaging     LifecycleState = "staging"
	LifecycleInterrupted LifecycleState = "interrupted"
	LifecycleFailed      LifecycleState = "failed"
	LifecycleCompleted   LifecycleState = "completed"
	LifecycleSuperseded  LifecycleState = "superseded"
	LifecycleRepaired    LifecycleState = "repaired"
	LifecycleDeleted     LifecycleState = "deleted"
)

func CanTransitionLifecycle(from, to LifecycleState) bool {
	allowed := map[LifecycleState]map[LifecycleState]bool{
		LifecycleStaging:     {LifecycleInterrupted: true, LifecycleFailed: true, LifecycleCompleted: true},
		LifecycleInterrupted: {LifecycleStaging: true, LifecycleFailed: true},
		LifecycleCompleted:   {LifecycleSuperseded: true, LifecycleRepaired: true, LifecycleDeleted: true},
		LifecycleRepaired:    {LifecycleSuperseded: true, LifecycleDeleted: true},
		LifecycleSuperseded:  {LifecycleDeleted: true},
		LifecycleFailed:      {LifecycleDeleted: true},
	}
	return from == to || allowed[from][to]
}

type EvidenceKind string

const (
	EvidenceSourceIdentity      EvidenceKind = "source_identity"
	EvidenceExtractedArtifact   EvidenceKind = "extracted_artifact"
	EvidenceGeneratedDerivative EvidenceKind = "generated_derivative"
	EvidenceToolReport          EvidenceKind = "tool_report"
	EvidenceAuthoredObservation EvidenceKind = "authored_observation"
	EvidenceInference           EvidenceKind = "inference"
)

type Authorship struct {
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	Method    string    `json:"method"`
}
type EvidenceRecord struct {
	ID             EvidenceID   `json:"id"`
	Kind           EvidenceKind `json:"kind"`
	Inputs         []string     `json:"inputs,omitempty"`
	Transformation string       `json:"transformation,omitempty"`
	Supports       []string     `json:"supports,omitempty"`
	Authorship     *Authorship  `json:"authorship,omitempty"`
}

type CustodyPolicy string

const (
	CustodyReferenceInPlace CustodyPolicy = "reference_in_place"
	CustodyCopyManaged      CustodyPolicy = "copy_managed"
	CustodyMoveManaged      CustodyPolicy = "move_managed"
	CustodyIdentityOnly     CustodyPolicy = "identity_only"
)

type PublicationState string

const (
	PublicationStaging   PublicationState = "staging"
	PublicationPublished PublicationState = "published"
)

type LineageRelation string

const (
	LineageRetryOf           LineageRelation = "retry_of"
	LineageResumedFrom       LineageRelation = "resumed_from"
	LineageExtendsCoverageOf LineageRelation = "extends_coverage_of"
	LineageSupersedes        LineageRelation = "supersedes"
)

type RunLineage struct {
	Relation LineageRelation `json:"relation"`
	RunID    RunID           `json:"run_id"`
}

type KnowledgeState string

const (
	KnowledgeUnknown           KnowledgeState = "unknown"
	KnowledgeKnownPresent      KnowledgeState = "known_present"
	KnowledgeKnownAbsent       KnowledgeState = "known_absent"
	KnowledgeNoneObserved      KnowledgeState = "none_observed"
	KnowledgeNotRequested      KnowledgeState = "not_requested"
	KnowledgeInapplicable      KnowledgeState = "inapplicable"
	KnowledgeFailedToDetermine KnowledgeState = "failed_to_determine"
)

type TimeSpan struct{ Start, End time.Duration }
type MediaTimestamp struct {
	Timeline, TimeBase   string
	Requested, Effective time.Duration
}
type CoverageGap struct {
	Span   TimeSpan
	State  KnowledgeState
	Reason string
}
type TranscriptCoverage struct {
	Attempted              TimeSpan
	Represented            []TimeSpan
	Completion, Confidence KnowledgeState
}
type VideoCoverage struct{ SampledInstants []MediaTimestamp }
type Coverage struct {
	Transcript TranscriptCoverage
	Video      VideoCoverage
	KnownGaps  []CoverageGap
}

type Manifest struct {
	SchemaVersion string           `json:"schema_version"`
	RunID         RunID            `json:"run_id,omitempty"`
	BundleID      BundleID         `json:"bundle_id,omitempty"`
	SourceID      SourceID         `json:"source_id,omitempty"`
	Lifecycle     LifecycleState   `json:"lifecycle"`
	Custody       CustodyPolicy    `json:"custody,omitempty"`
	Publication   PublicationState `json:"publication,omitempty"`
	Lineage       []RunLineage     `json:"lineage,omitempty"`
	Evidence      []EvidenceRecord `json:"evidence,omitempty"`
	Coverage      Coverage         `json:"coverage,omitzero"`
}

func (m Manifest) Validate() error {
	seen := make(map[string]bool, len(m.Evidence))
	for _, evidence := range m.Evidence {
		id := string(evidence.ID)
		if id == "" || seen[id] {
			return errors.New("evidence identifiers must be non-empty and unique")
		}
		seen[id] = true
		if evidence.Kind == EvidenceExtractedArtifact || evidence.Kind == EvidenceGeneratedDerivative {
			if len(evidence.Inputs) == 0 || evidence.Transformation == "" {
				return errors.New("derivatives require immediate inputs and transformation")
			}
		}
		if evidence.Kind == EvidenceAuthoredObservation || evidence.Kind == EvidenceInference {
			if evidence.Authorship == nil || evidence.Authorship.Author == "" || evidence.Authorship.Method == "" || evidence.Authorship.CreatedAt.IsZero() || len(evidence.Supports) == 0 {
				return errors.New("authored evidence requires authorship, method, time, and support")
			}
		}
	}
	for _, evidence := range m.Evidence {
		for _, input := range append(append([]string{}, evidence.Inputs...), evidence.Supports...) {
			if !seen[input] {
				return errors.New("evidence reference not found")
			}
		}
	}
	return nil
}

type ResourcePolicy struct {
	MaxProcesses   int
	MaxMemoryBytes int64
	CPUTime        time.Duration
}
type StreamPolicy struct{ CaptureStdout, CaptureStderr bool }
type CommandSpec struct {
	Executable       string
	Args             []string
	Environment      map[string]string
	WorkingDirectory string
	Streams          StreamPolicy
	Deadline         time.Duration
	LogLimitBytes    int64
	ResourcePolicy   ResourcePolicy
}
type CommandResult struct {
	Started             bool
	Stdout, Stderr      []byte
	ExitCode            int
	Signal              string
	TimedOut, Cancelled bool
	Diagnostics         []StructuredError
	ArtifactValidations []ArtifactValidation
}
type ArtifactValidation struct {
	Path          string
	Valid         bool
	Digest, Error string
}
type ProcessRunner interface {
	Run(context.Context, CommandSpec) (CommandResult, error)
}

type Operation string

const (
	OperationInspect    Operation = "inspect"
	OperationSample     Operation = "sample"
	OperationTranscribe Operation = "transcribe"
)

type ExecutableIdentity struct{ Path, Version, Digest string }
type CapabilityRequest struct {
	Operation        Operation
	AllowSideEffects bool
}
type CapabilityReport struct {
	Operation         Operation
	Executable        ExecutableIdentity
	Supported         bool
	SideEffectBounded bool
	Details           map[string]string
}
type CapabilityProvider interface {
	Discover(context.Context, CapabilityRequest) (CapabilityReport, error)
}

type InputMode string

const (
	InputPreparedAudio InputMode = "prepared-audio"
	InputOriginalMedia InputMode = "original-media"
)

type TimestampOrigin string

const TimestampMediaStart TimestampOrigin = "media-start"

type ConfidenceSemantics string

const (
	ConfidenceUnavailable ConfidenceSemantics = "unavailable"
	ConfidenceOptional    ConfidenceSemantics = "optional"
)

type TrustClass string

const (
	TrustLocal             TrustClass = "local"
	TrustExperimentalLocal TrustClass = "experimental-local"
	TrustHosted            TrustClass = "hosted"
)
const RedactedValue = "[REDACTED]"

type TranscriptionCapabilities struct {
	InputModes               []InputMode
	TimestampOrigin          TimestampOrigin
	Streaming, PartialOutput bool
	Confidence               ConfidenceSemantics
	RequiresModel, UsesCache bool
	Trust                    TrustClass
}
type AdapterDeclaration struct {
	InputModes               []InputMode
	TimestampOrigin          TimestampOrigin
	Streaming, PartialOutput bool
	Confidence               ConfidenceSemantics
	RequiresModel, UsesCache bool
	TrustClass               TrustClass
}
type EngineIdentity struct{ Name, Version string }
type TranscriptionRequest struct {
	InputPath, Language string
	Config              map[string]string
}
type TranscriptSegment struct {
	Start, End time.Duration
	Text       string
	Confidence *float64
}
type TranscriptionResult struct {
	Segments        []TranscriptSegment
	Engine          EngineIdentity
	Language        string
	EffectiveConfig map[string]string
}
type TranscriptResult = TranscriptionResult
type TranscriptionAdapter interface {
	Declaration() AdapterDeclaration
	Transcribe(context.Context, TranscriptionRequest) (TranscriptionResult, error)
}
