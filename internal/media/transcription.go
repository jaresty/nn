package media

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Transcriber interface {
	Capabilities() TranscriptionCapabilities
	Transcribe(context.Context, TranscriptionRequest) (TranscriptResult, error)
}
type ParakeetAdapter struct {
	executable string
	runner     ProcessRunner
}

func NewParakeetAdapter(executable string, runner ProcessRunner) *ParakeetAdapter {
	return &ParakeetAdapter{executable: executable, runner: runner}
}
func (*ParakeetAdapter) Declaration() AdapterDeclaration {
	return AdapterDeclaration{InputModes: []InputMode{InputPreparedAudio}, TimestampOrigin: TimestampMediaStart, Confidence: ConfidenceOptional, RequiresModel: true, UsesCache: true, TrustClass: TrustExperimentalLocal}
}
func (p *ParakeetAdapter) Capabilities() TranscriptionCapabilities {
	d := p.Declaration()
	return TranscriptionCapabilities{InputModes: d.InputModes, TimestampOrigin: d.TimestampOrigin, Streaming: d.Streaming, PartialOutput: d.PartialOutput, Confidence: d.Confidence, RequiresModel: d.RequiresModel, UsesCache: d.UsesCache, Trust: d.TrustClass}
}

type parakeetJSON struct {
	Duration      float64           `json:"duration"`
	InferenceTime float64           `json:"inference_time"`
	Text          string            `json:"text"`
	Language      string            `json:"language"`
	Segments      []parakeetSegment `json:"segments"`
	Sentences     []parakeetSegment `json:"sentences"`
}
type parakeetSegment struct {
	Start      float64  `json:"start"`
	End        float64  `json:"end"`
	Text       string   `json:"text"`
	Confidence *float64 `json:"confidence,omitempty"`
}

func (p *ParakeetAdapter) Transcribe(ctx context.Context, request TranscriptionRequest) (TranscriptResult, error) {
	if p.runner == nil {
		return TranscriptResult{}, fmt.Errorf("parakeet runner is not configured")
	}
	tempDir, err := os.MkdirTemp("", "nn-parakeet-")
	if err != nil {
		return TranscriptResult{}, fmt.Errorf("create parakeet workspace: %w", err)
	}
	defer os.RemoveAll(tempDir)
	prepared := filepath.Join(tempDir, "prepared.wav")
	prepare := CommandSpec{Executable: "ffmpeg", Args: []string{"-v", "error", "-i", request.InputPath, "-vn", "-ac", "1", "-ar", "16000", "-c:a", "pcm_s16le", "-y", prepared}, Streams: StreamPolicy{CaptureStderr: true}}
	prepareResult, err := p.runner.Run(ctx, prepare)
	if err != nil {
		return TranscriptResult{}, fmt.Errorf("prepare parakeet audio: %w", err)
	}
	if prepareResult.ExitCode != 0 {
		return TranscriptResult{}, fmt.Errorf("prepare parakeet audio: ffmpeg exited with code %d", prepareResult.ExitCode)
	}
	result, err := p.runner.Run(ctx, CommandSpec{Executable: p.executable, Args: []string{"transcribe", prepared, "--format", "json"}, Streams: StreamPolicy{CaptureStdout: true, CaptureStderr: true}})
	if err != nil {
		return TranscriptResult{}, fmt.Errorf("run parakeet: %w", err)
	}
	if result.ExitCode != 0 {
		return TranscriptResult{}, fmt.Errorf("parakeet exited with code %d", result.ExitCode)
	}
	var document parakeetJSON
	if err := json.Unmarshal(result.Stdout, &document); err != nil {
		return TranscriptResult{}, fmt.Errorf("decode parakeet output: %w", err)
	}
	segments := document.Segments
	if len(segments) == 0 {
		segments = document.Sentences
	}
	timing := "native-segment-boundaries"
	if len(segments) == 0 && strings.TrimSpace(document.Text) != "" && document.Duration > 0 {
		segments = []parakeetSegment{{Start: 0, End: document.Duration, Text: strings.TrimSpace(document.Text)}}
		timing = "synthesized-document-boundaries"
	}
	out := TranscriptResult{Engine: EngineIdentity{Name: "parakeet"}, Language: document.Language, EffectiveConfig: map[string]string{"input": "prepared-16khz-mono-pcm-wav", "format": "json", "trust": "experimental-local", "timestamps": timing, "inference_time_seconds": strconv.FormatFloat(document.InferenceTime, 'f', -1, 64)}}
	for _, segment := range segments {
		if strings.TrimSpace(segment.Text) == "" || segment.End <= segment.Start {
			continue
		}
		out.Segments = append(out.Segments, TranscriptSegment{Start: time.Duration(segment.Start * float64(time.Second)), End: time.Duration(segment.End * float64(time.Second)), Text: strings.TrimSpace(segment.Text), Confidence: segment.Confidence})
	}
	if len(out.Segments) == 0 {
		return TranscriptResult{}, fmt.Errorf("decode parakeet output: no transcript segments")
	}
	return out, nil
}
