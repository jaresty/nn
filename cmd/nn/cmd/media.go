package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

const mediaResultVersion = "nn.media.run-result/v1"

type mediaStageResult struct {
	Name      string `json:"name,omitempty"`
	Requested bool   `json:"requested"`
	State     string `json:"state"`
}
type mediaStructuredError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Stage   string `json:"stage,omitempty"`
}
type mediaCommandResult struct {
	SchemaVersion   string                 `json:"schema_version"`
	Command         string                 `json:"command,omitempty"`
	RunID           string                 `json:"run_id,omitempty"`
	SourceID        string                 `json:"source_id,omitempty"`
	BundleID        string                 `json:"bundle_id,omitempty"`
	ManifestLocator string                 `json:"manifest_locator,omitempty"`
	Outcome         string                 `json:"outcome"`
	Stages          []mediaStageResult     `json:"stages,omitempty"`
	Warnings        []string               `json:"warnings,omitempty"`
	Errors          []mediaStructuredError `json:"errors,omitempty"`
}
type mediaCommandRequest struct {
	Operation, Source, Every, Engine, RunID, Install string
	Frames                                           []string
	ContactSheet, NonInteractive, Confirm            bool
}
type mediaCommandService interface {
	Execute(context.Context, mediaCommandRequest) (mediaCommandResult, error)
	Discover(context.Context, string) (mediaCommandResult, error)
	Context(context.Context, string, int, int) (any, error)
	Doctor(context.Context, mediaCommandRequest) (mediaCommandResult, error)
}
type unavailableMediaService struct{}

func (unavailableMediaService) unavailable(command string) mediaCommandResult {
	return mediaCommandResult{SchemaVersion: mediaResultVersion, Command: command, Outcome: "prerequisite_failed", Errors: []mediaStructuredError{{Code: "media_service_unavailable", Message: "media pipeline is unavailable; run nn media doctor"}}}
}
func (s unavailableMediaService) Execute(_ context.Context, r mediaCommandRequest) (mediaCommandResult, error) {
	return s.unavailable(r.Operation), nil
}
func (s unavailableMediaService) Discover(_ context.Context, id string) (mediaCommandResult, error) {
	x := s.unavailable("runs")
	x.RunID = id
	return x, nil
}
func (s unavailableMediaService) Context(context.Context, string, int, int) (any, error) {
	return nil, fmt.Errorf("media service unavailable")
}
func (s unavailableMediaService) Doctor(_ context.Context, _ mediaCommandRequest) (mediaCommandResult, error) {
	return s.unavailable("doctor"), nil
}

func newMediaCmd(_ *rootState, service mediaCommandService) *cobra.Command {
	if service == nil {
		service = unavailableMediaService{}
	}
	var jsonMode bool
	cmd := &cobra.Command{Use: "media", Short: "Prepare and retrieve provenance-preserving media evidence"}
	cmd.PersistentFlags().BoolVar(&jsonMode, "json", false, "Emit one versioned JSON result")
	render := func(c *cobra.Command, result mediaCommandResult) error {
		if result.SchemaVersion == "" {
			result.SchemaVersion = mediaResultVersion
		}
		if jsonMode {
			return json.NewEncoder(outWriter(c)).Encode(result)
		}
		w := outWriter(c)
		fmt.Fprintf(w, "%s: %s\n", result.Command, result.Outcome)
		if result.RunID != "" {
			fmt.Fprintf(w, "run: %s\n", result.RunID)
		}
		if result.ManifestLocator != "" {
			fmt.Fprintf(w, "manifest: %s\n", result.ManifestLocator)
		}
		for _, s := range result.Stages {
			fmt.Fprintf(w, "stage %s: %s\n", s.Name, s.State)
		}
		for _, e := range result.Errors {
			fmt.Fprintf(w, "error %s: %s\n", e.Code, e.Message)
		}
		return nil
	}
	add := func(name string, configure func(*cobra.Command, *mediaCommandRequest)) {
		r := mediaCommandRequest{Operation: name}
		child := &cobra.Command{Use: name + " <file>", Args: cobra.ExactArgs(1), RunE: func(c *cobra.Command, args []string) error {
			r.Source = args[0]
			r.NonInteractive = !isTTYFn()
			result, err := service.Execute(c.Context(), r)
			if err != nil {
				return err
			}
			if result.Command == "" {
				result.Command = name
			}
			return render(c, result)
		}}
		if configure != nil {
			configure(child, &r)
		}
		cmd.AddCommand(child)
	}
	samplingFlags := func(c *cobra.Command, r *mediaCommandRequest) {
		c.Flags().StringVar(&r.Every, "every", "", "Sampling interval")
		c.Flags().StringSliceVar(&r.Frames, "frames", nil, "Explicit frame timestamps")
		c.Flags().BoolVar(&r.ContactSheet, "contact-sheet", false, "Generate a contact sheet")
	}
	add("inspect", nil)
	add("sample", samplingFlags)
	add("transcribe", func(c *cobra.Command, r *mediaCommandRequest) {
		c.Flags().StringVar(&r.Engine, "engine", "", "Explicit transcription engine")
	})
	add("prepare", func(c *cobra.Command, r *mediaCommandRequest) {
		samplingFlags(c, r)
		c.Flags().StringVar(&r.Engine, "engine", "", "Optional explicit transcription engine")
	})
	var runID string
	var maxBytes, page int
	contextCmd := &cobra.Command{Use: "context --run RUN_ID", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		if runID == "" {
			return fmt.Errorf("--run is required")
		}
		packet, err := service.Context(c.Context(), runID, maxBytes, page)
		if err != nil {
			return err
		}
		enc := json.NewEncoder(outWriter(c))
		enc.SetEscapeHTML(false)
		return enc.Encode(packet)
	}}
	contextCmd.Flags().StringVar(&runID, "run", "", "Existing run ID")
	contextCmd.Flags().IntVar(&maxBytes, "max-bytes", 32768, "Maximum transcript text bytes per packet page")
	contextCmd.Flags().IntVar(&page, "page", 1, "Transcript packet page")
	cmd.AddCommand(contextCmd)
	var doctor mediaCommandRequest
	doctor.Operation = "doctor"
	doctorCmd := &cobra.Command{Use: "doctor", Args: cobra.NoArgs, RunE: func(c *cobra.Command, _ []string) error {
		doctor.NonInteractive = !isTTYFn()
		if doctor.Install != "" && !isTTYFn() && !doctor.Confirm {
			return fmt.Errorf("non-TTY install requires --confirm")
		}
		result, err := service.Doctor(c.Context(), doctor)
		if err != nil {
			return err
		}
		return render(c, result)
	}}
	doctorCmd.Flags().StringVar(&doctor.Install, "install", "", "Select dependency diagnostics")
	doctorCmd.Flags().BoolVar(&doctor.Confirm, "confirm", false, "Confirm remediation")
	cmd.AddCommand(doctorCmd)
	cmd.AddCommand(&cobra.Command{Use: "runs <run-id>", Args: cobra.ExactArgs(1), RunE: func(c *cobra.Command, args []string) error {
		result, err := service.Discover(c.Context(), strings.TrimSpace(args[0]))
		if err != nil {
			return err
		}
		return render(c, result)
	}})
	return cmd
}
