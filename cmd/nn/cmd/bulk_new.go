package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

// bulkNewSpec is the JSON schema for each note in a bulk-new batch.
type bulkNewSpec struct {
	Title   string          `json:"title"`
	Type    string          `json:"type"`
	Content string          `json:"content"`
	Tags    []string        `json:"tags"`
	Links   []bulkNewLink   `json:"links"`
}

// bulkNewLink declares a link from this note to another note in the batch by index.
type bulkNewLink struct {
	Ref        int    `json:"ref"`        // index into the batch array
	Annotation string `json:"annotation"`
	Type       string `json:"type"`
}

func newBulkNewCmd(state *rootState) *cobra.Command {
	var jsonInput string

	cmd := &cobra.Command{
		Use:   "bulk-new --json <json-array>",
		Short: "Create multiple notes with inline links in a single commit",
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonInput == "" {
				return fmt.Errorf("--json is required")
			}

			var specs []bulkNewSpec
			if err := json.Unmarshal([]byte(jsonInput), &specs); err != nil {
				return fmt.Errorf("bulk-new: parse JSON: %w", err)
			}
			if len(specs) == 0 {
				return fmt.Errorf("bulk-new: empty batch")
			}

			now := time.Now().UTC()
			notes := make([]*note.Note, len(specs))
			for i, s := range specs {
				if s.Title == "" {
					return fmt.Errorf("bulk-new: spec[%d] missing title", i)
				}
				typ := note.Type(s.Type)
				if !typ.IsValid() {
					return fmt.Errorf("bulk-new: spec[%d] invalid type %q", i, s.Type)
				}
				var parsedTags []string
				for _, t := range s.Tags {
					if t = strings.TrimSpace(t); t != "" {
						parsedTags = append(parsedTags, t)
					}
				}
				notes[i] = &note.Note{
					ID:       note.GenerateID(),
					Title:    s.Title,
					Type:     typ,
					Status:   note.StatusDraft,
					Tags:     parsedTags,
					Created:  now,
					Modified: now,
					Body:     s.Content,
				}
			}

			// Resolve inline links using ref indices.
			for i, s := range specs {
				for _, lnk := range s.Links {
					if lnk.Ref < 0 || lnk.Ref >= len(notes) {
						return fmt.Errorf("bulk-new: spec[%d] link ref %d out of range", i, lnk.Ref)
					}
					if lnk.Annotation == "" {
						return fmt.Errorf("bulk-new: spec[%d] link to ref %d missing annotation", i, lnk.Ref)
					}
					if !note.IsKnownLinkType(lnk.Type) {
						fmt.Fprintf(cmd.ErrOrStderr(), "warning: unknown link type %q in spec[%d]\n", lnk.Type, i)
					}
					notes[i].Links = append(notes[i].Links, note.Link{
						TargetID:   notes[lnk.Ref].ID,
						Annotation: lnk.Annotation,
						Type:       lnk.Type,
					})
				}
			}

			if err := state.backend.BulkWrite(notes); err != nil {
				return fmt.Errorf("bulk-new: %w", err)
			}

			w := outWriter(cmd)
			for _, n := range notes {
				fmt.Fprintf(w, "created %s\n", n.ID)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&jsonInput, "json", "", "JSON array of note specs")
	return cmd
}
