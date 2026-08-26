package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/jaresty/nn/internal/note"
)

type applyNoteSpec struct {
	Key         string   `yaml:"key"`
	Title       string   `yaml:"title"`
	Type        string   `yaml:"type"`
	Content     string   `yaml:"content"`
	Tags        []string `yaml:"tags"`
	AppliesWhen string   `yaml:"applies_when"`
}

type applyEdgeSpec struct {
	From       string `yaml:"from"`
	To         string `yaml:"to"`
	Type       string `yaml:"type"`
	Annotation string `yaml:"annotation"`
}

type applyManifest struct {
	Notes []applyNoteSpec `yaml:"notes"`
	Edges []applyEdgeSpec `yaml:"edges"`
}

func newGraphApplyCmd(state *rootState) *cobra.Command {
	var dryRun bool
	var commit bool

	cmd := &cobra.Command{
		Use:   "apply <manifest.yaml>",
		Short: "Apply a YAML changeset manifest (create notes and edges atomically)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !dryRun && !commit {
				return fmt.Errorf("graph apply: one of --dry-run or --commit is required")
			}

			data, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("graph apply: read manifest: %w", err)
			}

			var manifest applyManifest
			if err := yaml.Unmarshal(data, &manifest); err != nil {
				return fmt.Errorf("graph apply: parse manifest: %w", err)
			}

			now := time.Now().UTC()
			keyMap := make(map[string]string) // key → note ID

			// Build notes.
			notes := make([]*note.Note, len(manifest.Notes))
			for i, s := range manifest.Notes {
				if s.Title == "" {
					return fmt.Errorf("graph apply: notes[%d] missing title", i)
				}
				typ := note.Type(s.Type)
				if !typ.IsValid() {
					return fmt.Errorf("graph apply: notes[%d] invalid type %q", i, s.Type)
				}
				id := note.GenerateID()
				notes[i] = &note.Note{
					ID:          id,
					Title:       s.Title,
					Type:        typ,
					Status:      note.StatusDraft,
					Tags:        s.Tags,
					Created:     now,
					Modified:    now,
					Body:        s.Content,
					AppliesWhen: s.AppliesWhen,
				}
				if s.Key != "" {
					if _, dup := keyMap[s.Key]; dup {
						return fmt.Errorf("graph apply: duplicate key %q", s.Key)
					}
					keyMap[s.Key] = id
				}
			}

			// Resolve edge references.
			type resolvedEdge struct {
				fromID     string
				toID       string
				linkType   string
				annotation string
			}
			edges := make([]resolvedEdge, 0, len(manifest.Edges))
			for i, e := range manifest.Edges {
				fromID, err := resolveRef(e.From, keyMap)
				if err != nil {
					return fmt.Errorf("graph apply: edges[%d].from: %w", i, err)
				}
				toID, err := resolveRef(e.To, keyMap)
				if err != nil {
					return fmt.Errorf("graph apply: edges[%d].to: %w", i, err)
				}
				if e.Annotation == "" {
					return fmt.Errorf("graph apply: edges[%d] missing annotation", i)
				}
				if !note.IsKnownLinkType(e.Type) {
					return fmt.Errorf("graph apply: edges[%d] has invalid type %q", i, e.Type)
				}
				edges = append(edges, resolvedEdge{
					fromID:     fromID,
					toID:       toID,
					linkType:   e.Type,
					annotation: e.Annotation,
				})
			}

			w := outWriter(cmd)

			if dryRun {
				fmt.Fprintf(w, "dry-run: would create %d note(s), add %d edge(s)\n", len(notes), len(edges))
				for _, n := range notes {
					fmt.Fprintf(w, "  create %s — %q\n", n.Type, n.Title)
				}
				for _, e := range edges {
					fmt.Fprintf(w, "  edge  %s -[%s]-> %s\n", e.fromID, e.linkType, e.toID)
				}
				return nil
			}

			// Attach new-batch edges to their source note objects.
			newIDToNote := make(map[string]*note.Note, len(notes))
			for _, n := range notes {
				newIDToNote[n.ID] = n
			}
			for _, e := range edges {
				if src, ok := newIDToNote[e.fromID]; ok {
					src.Links = append(src.Links, note.Link{
						TargetID:   e.toID,
						Type:       e.linkType,
						Annotation: e.annotation,
						Status:     "draft",
					})
				}
			}

			// Read and mutate existing-source notes so all writes land in one commit.
			existingUpdates := make(map[string]*note.Note)
			for _, e := range edges {
				if newIDToNote[e.fromID] != nil {
					continue // handled above
				}
				src, ok := existingUpdates[e.fromID]
				if !ok {
					var err error
					src, err = state.backend.Read(e.fromID)
					if err != nil {
						return fmt.Errorf("graph apply: read existing note %s: %w", e.fromID, err)
					}
					existingUpdates[e.fromID] = src
				}
				src.Links = append(src.Links, note.Link{
					TargetID:   e.toID,
					Type:       e.linkType,
					Annotation: e.annotation,
					Status:     "draft",
				})
			}
			updateNotes := make([]*note.Note, 0, len(existingUpdates))
			for _, n := range existingUpdates {
				updateNotes = append(updateNotes, n)
			}

			if err := state.backend.BulkApply(notes, updateNotes); err != nil {
				return fmt.Errorf("graph apply: %w", err)
			}

			for _, n := range notes {
				fmt.Fprintf(w, "created %s — %q\n", n.ID, n.Title)
			}
			fmt.Fprintf(w, "applied %d edge(s)\n", len(edges))
			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print planned changes without writing")
	cmd.Flags().BoolVar(&commit, "commit", false, "Apply changes atomically in one git commit")
	return cmd
}

// resolveRef resolves a manifest reference to a note ID.
// Supports temporary keys and "existing:<id>" syntax.
func resolveRef(ref string, keyMap map[string]string) (string, error) {
	if strings.HasPrefix(ref, "existing:") {
		id := strings.TrimPrefix(ref, "existing:")
		if id == "" {
			return "", fmt.Errorf("empty id after existing: prefix")
		}
		return id, nil
	}
	id, ok := keyMap[ref]
	if !ok {
		known := make([]string, 0, len(keyMap))
		for k := range keyMap {
			known = append(known, k)
		}
		return "", fmt.Errorf("unknown key %q (known keys: %s)", ref, strings.Join(known, ", "))
	}
	return id, nil
}
