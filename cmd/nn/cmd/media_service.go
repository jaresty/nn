package cmd

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	im "github.com/jaresty/nn/internal/media"
)

type productionMediaService struct {
	root   string
	store  *im.LocalRunStore
	runner im.ProcessRunner
}

func newProductionMediaService() mediaCommandService {
	root := os.Getenv("NN_MEDIA_ROOT")
	if root == "" {
		if d, err := os.UserConfigDir(); err == nil {
			root = filepath.Join(d, "nn", "media")
		} else {
			root = filepath.Join(os.TempDir(), "nn-media")
		}
	}
	return &productionMediaService{root: root, store: im.NewLocalRunStore(root), runner: im.OSProcessRunner{}}
}
func newProductionMediaServiceAt(root string, runner im.ProcessRunner) mediaCommandService {
	return &productionMediaService{root: root, store: im.NewLocalRunStore(root), runner: runner}
}
func mediaRunID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return time.Now().UTC().Format("20060102T150405.000000000") + "-" + hex.EncodeToString(b[:])
}
func sourceID(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
func commandResult(r im.RunResult) mediaCommandResult {
	out := mediaCommandResult{SchemaVersion: r.SchemaVersion, Command: r.Command, RunID: string(r.RunID), SourceID: string(r.SourceID), BundleID: string(r.BundleID), ManifestLocator: r.ManifestLocator, Outcome: string(r.Outcome), NoteCapture: string(r.NoteCapture), Warnings: r.Warnings}
	for _, s := range r.Stages {
		out.Stages = append(out.Stages, mediaStageResult{Name: s.Name, Requested: s.Requested, State: string(s.State)})
	}
	for _, e := range r.Errors {
		out.Errors = append(out.Errors, mediaStructuredError{Code: e.Code, Message: e.Message, Stage: e.Stage})
	}
	return out
}
func (s *productionMediaService) persist(ctx context.Context, command, source string, stages []im.StageResult, projection im.NoteProjection, transcript []im.TranscriptSegment, images []im.ImageAttachment) (mediaCommandResult, error) {
	id := mediaRunID()
	sid, err := sourceID(source)
	if err != nil {
		return mediaCommandResult{}, err
	}
	result := im.RunResult{SchemaVersion: im.RunResultSchemaVersion, Command: command, RunID: im.RunID(id), BundleID: im.BundleID(id), SourceID: im.SourceID(sid), Stages: stages, Outcome: im.DeriveOutcome(stages, im.StageNotRequested), NoteCapture: im.StageNotRequested}
	evidence := []im.EvidenceRecord{{ID: "source", Kind: im.EvidenceSourceIdentity}}
	if strings.Contains(projection.Markdown, "## Transcript") {
		evidence = append(evidence, im.EvidenceRecord{ID: "transcript", Kind: im.EvidenceToolReport, Inputs: []string{"source"}, Transformation: "experimental Parakeet transcript projection"})
	}
	artifactNumber := 0
	inArtifacts := false
	for _, line := range strings.Split(projection.Markdown, "\n") {
		if line == "## Artifacts" {
			inArtifacts = true
			continue
		}
		if strings.HasPrefix(line, "## ") {
			inArtifacts = false
		}
		if !inArtifacts {
			continue
		}
		start, end := strings.Index(line, "`"), strings.LastIndex(line, "`")
		if start < 0 || end <= start {
			continue
		}
		artifactNumber++
		evidence = append(evidence, im.EvidenceRecord{ID: im.EvidenceID(fmt.Sprintf("artifact-%d", artifactNumber)), Kind: im.EvidenceGeneratedDerivative, Inputs: []string{"source"}, Transformation: strings.TrimSpace(line[:start]) + " path=" + line[start+1:end]})
	}
	coverage := im.Coverage{Transcript: im.TranscriptCoverage{Completion: im.KnowledgeNotRequested, Confidence: im.KnowledgeUnknown}}
	if len(transcript) > 0 {
		coverage.Transcript.Completion = im.KnowledgeKnownPresent
		coverage.Transcript.Represented = []im.TimeSpan{{Start: transcript[0].Start, End: transcript[len(transcript)-1].End}}
	}
	for _, image := range images {
		if image.Timestamp != nil {
			coverage.Video.SampledInstants = append(coverage.Video.SampledInstants, *image.Timestamp)
		}
	}
	if len(coverage.Video.SampledInstants) > 0 {
		coverage.KnownGaps = append(coverage.KnownGaps, im.CoverageGap{State: im.KnowledgeUnknown, Reason: "sampled instants do not cover intervening video intervals"})
	}
	manifest := im.Manifest{SchemaVersion: im.ManifestSchemaVersion, RunID: result.RunID, BundleID: result.BundleID, SourceID: result.SourceID, Lifecycle: im.LifecycleCompleted, Custody: im.CustodyReferenceInPlace, Publication: im.PublicationPublished, Evidence: evidence, Coverage: coverage}
	if err := s.store.Publish(ctx, im.StoredRun{Result: result, Manifest: manifest, Projection: projection, Transcript: transcript, Images: images}); err != nil {
		return mediaCommandResult{}, err
	}
	stored, err := s.store.Discover(ctx, id)
	if err != nil {
		return mediaCommandResult{}, err
	}
	return commandResult(stored.Result), nil
}
func parseFrameTimestamp(value string) (time.Duration, error) {
	if d, err := time.ParseDuration(value); err == nil {
		return d, nil
	}
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid frame timestamp %q", value)
	}
	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid frame timestamp %q", value)
	}
	minutes, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid frame timestamp %q", value)
	}
	seconds, err := strconv.ParseFloat(parts[2], 64)
	if err != nil || minutes < 0 || minutes >= 60 || seconds < 0 || seconds >= 60 {
		return 0, fmt.Errorf("invalid frame timestamp %q", value)
	}
	return time.Duration(hours)*time.Hour + time.Duration(minutes)*time.Minute + time.Duration(seconds*float64(time.Second)), nil
}

func (s *productionMediaService) sampling(ctx context.Context, r mediaCommandRequest, duration time.Duration) (string, []im.ImageAttachment, error) {
	interval := 45 * time.Second
	var err error
	if r.Every != "" {
		interval, err = time.ParseDuration(r.Every)
		if err != nil {
			return "", nil, fmt.Errorf("invalid --every: %w", err)
		}
	}
	explicit := make([]time.Duration, 0, len(r.Frames))
	for _, value := range r.Frames {
		timestamp, parseErr := parseFrameTimestamp(value)
		if parseErr != nil {
			return "", nil, parseErr
		}
		explicit = append(explicit, timestamp)
	}
	plan, err := im.BuildSamplingPlan(duration, interval, explicit)
	if err != nil {
		return "", nil, err
	}
	dir := filepath.Join(s.root, "artifacts", mediaRunID())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", nil, err
	}
	artifacts, err := im.ExtractFrames(ctx, s.runner, r.Source, plan.Timestamps, dir)
	if err != nil {
		return "", nil, err
	}
	lines := []string{"## Coverage", "", "Sampled instants only; explicit timestamps augment interval samples. Intervening intervals were not reviewed.", "", "## Artifacts", ""}
	images := make([]im.ImageAttachment, 0, len(artifacts)+1)
	for _, artifact := range artifacts {
		lines = append(lines, fmt.Sprintf("- Frame %s `%s`", im.FormatTimestamp(artifact.Timestamp), artifact.Path))
		timestamp := im.MediaTimestamp{Timeline: "source", TimeBase: "media", Requested: artifact.Timestamp, Effective: artifact.Timestamp}
		images = append(images, im.ImageAttachment{Path: artifact.Path, MIME: "image/jpeg", Timestamp: &timestamp, Role: "sampled-frame"})
	}
	if r.ContactSheet {
		path := filepath.Join(dir, "contact-sheet.jpg")
		if err := im.CreateContactSheet(ctx, s.runner, r.Source, duration, interval, 4, path); err != nil {
			return "", nil, err
		}
		lines = append(lines, fmt.Sprintf("- Contact sheet (interval %s) `%s`", interval, path))
		images = append(images, im.ImageAttachment{Path: path, MIME: "image/jpeg", Role: "contact-sheet"})
	}
	return strings.Join(lines, "\n") + "\n", images, nil
}

func (s *productionMediaService) Execute(ctx context.Context, r mediaCommandRequest) (mediaCommandResult, error) {
	switch r.Operation {
	case "inspect", "prepare", "sample":
		metadata, err := im.Inspect(ctx, s.runner, r.Source)
		if err != nil {
			return s.failed(r.Operation, "inspect_failed", err), nil
		}
		markdown := fmt.Sprintf("## Source and metadata\n\nSource: `%s`\nDuration: %s\nVideo: %s %dx%d\nAudio present: %t\n\n", r.Source, metadata.Duration, metadata.VideoCodec, metadata.Width, metadata.Height, metadata.HasAudio)
		stages := []im.StageResult{{Name: "inspect", Requested: true, State: im.StageSucceeded}}
		var transcript []im.TranscriptSegment
		var images []im.ImageAttachment
		wantsSampling := r.Operation == "sample" || r.Every != "" || len(r.Frames) > 0 || r.ContactSheet
		if wantsSampling {
			sampled, sampledImages, sampleErr := s.sampling(ctx, r, metadata.Duration)
			if sampleErr != nil {
				stages = append(stages, im.StageResult{Name: "sample", Requested: true, State: im.StageFailed})
				markdown += "## Coverage\n\nSampling failed: " + sampleErr.Error() + "\n"
			} else {
				stages = append(stages, im.StageResult{Name: "sample", Requested: true, State: im.StageSucceeded})
				markdown += sampled
				images = sampledImages
			}
		} else {
			markdown += "## Coverage\n\nMetadata inspection completed. No video interval is represented as reviewed.\n"
		}
		if r.Operation == "prepare" && r.Engine != "" {
			if r.Engine != "parakeet" {
				stages = append(stages, im.StageResult{Name: "transcribe", Requested: true, State: im.StageUnavailable})
			} else {
				adapter := im.NewParakeetAdapter("parakeet", s.runner)
				transcriptResult, transcribeErr := adapter.Transcribe(ctx, im.TranscriptionRequest{InputPath: r.Source})
				if transcribeErr != nil {
					stages = append(stages, im.StageResult{Name: "transcribe", Requested: true, State: im.StageFailed})
					markdown += "\nTranscript unavailable: " + transcribeErr.Error() + "\n"
				} else {
					stages = append(stages, im.StageResult{Name: "transcribe", Requested: true, State: im.StageSucceeded})
					if transcriptResult.EffectiveConfig["timestamps"] == "synthesized-document-boundaries" {
						markdown += "\nTranscript timing uses synthesized document-level boundaries spanning [0,duration], not native sentence timings.\n"
					}
					markdown += "\n## Transcript\n\n"
					for _, segment := range transcriptResult.Segments {
						markdown += fmt.Sprintf("- %s–%s %s\n", segment.Start, segment.End, segment.Text)
					}
					transcript = append(transcript, transcriptResult.Segments...)
				}
			}
		}
		title := "Media: " + filepath.Base(r.Source)
		if r.Operation == "sample" {
			title = "Media samples: " + filepath.Base(r.Source)
		}
		return s.persist(ctx, r.Operation, r.Source, stages, im.NoteProjection{Title: title, Markdown: markdown}, transcript, images)
	case "transcribe":
		if r.Engine != "parakeet" {
			return s.failed("transcribe", "explicit_engine_required", fmt.Errorf("select experimental engine with --engine parakeet")), nil
		}
		adapter := im.NewParakeetAdapter("parakeet", s.runner)
		tr, err := adapter.Transcribe(ctx, im.TranscriptionRequest{InputPath: r.Source})
		if err != nil {
			return s.failed("transcribe", "transcription_failed", err), nil
		}
		timing := "native timestamp boundaries"
		if tr.EffectiveConfig["timestamps"] == "synthesized-document-boundaries" {
			timing = "synthesized document-level boundaries spanning [0,duration], not native sentence timings"
		}
		markdown := "## Coverage\n\nTranscript generated by explicitly selected experimental Parakeet adapter using " + timing + "; semantic accuracy is not established.\n\n## Transcript\n\n"
		for _, seg := range tr.Segments {
			markdown += fmt.Sprintf("- %s–%s %s\n", seg.Start, seg.End, seg.Text)
		}
		return s.persist(ctx, "transcribe", r.Source, []im.StageResult{{Name: "transcribe", Requested: true, State: im.StageSucceeded}}, im.NoteProjection{Title: "Transcript: " + filepath.Base(r.Source), Markdown: markdown}, tr.Segments, nil)
	default:
		return mediaCommandResult{}, fmt.Errorf("unsupported media operation %q", r.Operation)
	}
}
func (s *productionMediaService) failed(command, code string, err error) mediaCommandResult {
	return mediaCommandResult{SchemaVersion: mediaResultVersion, Command: command, Outcome: string(im.OutcomeProcessingFailed), Stages: []mediaStageResult{{Name: command, Requested: true, State: string(im.StageFailed)}}, Errors: []mediaStructuredError{{Code: code, Message: err.Error(), Stage: command}}}
}
func (s *productionMediaService) Discover(ctx context.Context, id string) (mediaCommandResult, error) {
	run, err := s.store.Discover(ctx, id)
	if err != nil {
		return mediaCommandResult{}, err
	}
	out := commandResult(run.Result)
	out.Command = "runs"
	return out, nil
}
func (s *productionMediaService) Context(ctx context.Context, id string, maxBytes, page int) (any, error) {
	return s.store.Context(ctx, id, maxBytes, page)
}
func (s *productionMediaService) Project(ctx context.Context, id string) (mediaNoteProjection, mediaCommandResult, error) {
	p, err := s.store.Project(ctx, id)
	if err != nil {
		return mediaNoteProjection{}, mediaCommandResult{}, err
	}
	run, err := s.store.Discover(ctx, id)
	if err != nil {
		return mediaNoteProjection{}, mediaCommandResult{}, err
	}
	out := commandResult(run.Result)
	out.Command = "capture"
	out.NoteCapture = string(im.StageSucceeded)
	return mediaNoteProjection{Title: p.Title, Markdown: p.Markdown}, out, nil
}
func (s *productionMediaService) Doctor(_ context.Context, r mediaCommandRequest) (mediaCommandResult, error) {
	deps := []string{"ffprobe", "ffmpeg"}
	if r.Install != "" {
		deps = []string{r.Install}
	}
	out := mediaCommandResult{SchemaVersion: mediaResultVersion, Command: "doctor", Outcome: string(im.OutcomeSucceeded)}
	for _, dep := range deps {
		path, err := exec.LookPath(dep)
		state := im.StageSucceeded
		if err != nil {
			state = im.StageUnavailable
			out.Outcome = string(im.OutcomePrerequisiteFailed)
			out.Errors = append(out.Errors, mediaStructuredError{Code: "dependency_missing", Message: fmt.Sprintf("%s unavailable on %s/%s; install it explicitly with your package manager", dep, runtime.GOOS, runtime.GOARCH), Stage: dep})
		} else {
			out.Warnings = append(out.Warnings, dep+"="+path)
		}
		out.Stages = append(out.Stages, mediaStageResult{Name: dep, Requested: true, State: string(state)})
	}
	return out, nil
}
