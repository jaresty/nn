package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

func newCaptureCmd(state *rootState) *cobra.Command {
	var (
		title   string
		typ     string
		content string
		tags    string
	)

	cmd := &cobra.Command{
		Use:   "capture",
		Short: "Quickly capture raw material as a draft note",
		Long: `Streamlines the inbox→note pipeline for processing external material
(articles, quotes, observations). Creates a draft note of type 'observation'
(or --type) pre-populated with the supplied content.

The LLM then refines the note, extracts atomic sub-notes via 'nn new', and
runs 'nn suggest-links' on each.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				return fmt.Errorf("capture: --title is required")
			}

			noteType := note.Type(typ)
			if noteType == "" {
				noteType = note.TypeObservation
			}
			if !noteType.IsValid() {
				return fmt.Errorf("capture: unknown type %q", typ)
			}

			id := note.GenerateID()
			now := time.Now().UTC()
			n := &note.Note{
				ID:       id,
				Title:    title,
				Type:     noteType,
				Status:   note.StatusDraft,
				Created:  now,
				Modified: now,
				Body:     content,
			}
			if tags != "" {
				for _, t := range strings.Split(tags, ",") {
					if t = strings.TrimSpace(t); t != "" {
						n.Tags = append(n.Tags, t)
					}
				}
			}

			if err := state.backend.Write(n); err != nil {
				return fmt.Errorf("capture: %w", err)
			}

			fmt.Fprintf(outWriter(cmd), "%s\n", n.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "Note title (required)")
	cmd.Flags().StringVar(&typ, "type", "", "Note type (default: observation)")
	cmd.Flags().StringVar(&content, "content", "", "Raw content to capture as note body")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tags")
	return cmd
}
