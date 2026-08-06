package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

type bulkUpdateSpec struct {
	ID          string   `json:"id"`
	Title       string   `json:"title,omitempty"`
	Type        string   `json:"type,omitempty"`
	Content     string   `json:"content,omitempty"`
	Status      string   `json:"status,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	AppliesWhen string   `json:"applies_when,omitempty"`
}

func newBulkUpdateCmd(state *rootState) *cobra.Command {
	var jsonInput string

	cmd := &cobra.Command{
		Use:   "bulk-update --json <json-array>",
		Short: "Update multiple existing notes in a single commit",
		RunE: func(cmd *cobra.Command, args []string) error {
			if jsonInput == "" {
				return fmt.Errorf("--json is required")
			}

			var specs []bulkUpdateSpec
			if err := json.Unmarshal([]byte(jsonInput), &specs); err != nil {
				return fmt.Errorf("bulk-update: parse JSON: %w", err)
			}
			if len(specs) == 0 {
				return fmt.Errorf("bulk-update: empty batch")
			}

			now := time.Now().UTC()
			notes := make([]*note.Note, 0, len(specs))

			for i, s := range specs {
				if s.ID == "" {
					return fmt.Errorf("bulk-update: spec[%d] missing id", i)
				}
				n, err := state.backend.Read(s.ID)
				if err != nil {
					return fmt.Errorf("bulk-update: spec[%d] read %s: %w", i, s.ID, err)
				}

				if s.Title != "" {
					n.Title = s.Title
				}
				if s.Type != "" {
					typ := note.Type(s.Type)
					if !typ.IsValid() {
						return fmt.Errorf("bulk-update: spec[%d] invalid type %q", i, s.Type)
					}
					n.Type = typ
				}
				if s.Content != "" {
					n.Body = s.Content
				}
				if s.Status != "" {
					st := note.Status(s.Status)
					if !st.IsValid() {
						return fmt.Errorf("bulk-update: spec[%d] invalid status %q", i, s.Status)
					}
					n.Status = st
				}
				if s.Tags != nil {
					var cleaned []string
					for _, t := range s.Tags {
						if t = strings.TrimSpace(t); t != "" {
							cleaned = append(cleaned, t)
						}
					}
					n.Tags = cleaned
				}
				if s.AppliesWhen != "" {
					n.AppliesWhen = s.AppliesWhen
				}
				n.Modified = now
				notes = append(notes, n)
			}

			w := outWriter(cmd)
			for _, n := range notes {
				if err := state.backend.Update(n, nil); err != nil {
					return fmt.Errorf("bulk-update: update %s: %w", n.ID, err)
				}
				fmt.Fprintf(w, "updated %s\n", n.ID)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&jsonInput, "json", "", "JSON array of update specs")
	return cmd
}
