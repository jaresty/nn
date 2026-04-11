package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

func newNewCmd(state *rootState) *cobra.Command {
	var (
		title      string
		typ        string
		tags       string
		content    string
		noEdit     bool
		linkTo     string
		annotation string
	)

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new note",
		RunE: func(cmd *cobra.Command, args []string) error {
			if typ == "" {
				return fmt.Errorf("--type is required (concept|argument|model|hypothesis|observation)")
			}
			noteType := note.Type(typ)
			if !noteType.IsValid() {
				return fmt.Errorf("invalid --type %q: must be concept|argument|model|hypothesis|observation", typ)
			}

			var parsedTags []string
			if tags != "" {
				for _, t := range strings.Split(tags, ",") {
					if t = strings.TrimSpace(t); t != "" {
						parsedTags = append(parsedTags, t)
					}
				}
			}

			now := time.Now().UTC()
			n := &note.Note{
				ID:       note.GenerateID(),
				Title:    title,
				Type:     noteType,
				Status:   note.StatusDraft,
				Tags:     parsedTags,
				Created:  now,
				Modified: now,
				Body:     content,
			}

			if linkTo != "" {
				if annotation == "" {
					return fmt.Errorf("--annotation is required when using --link-to")
				}
				n.Links = []note.Link{{TargetID: linkTo, Annotation: annotation}}
			}

			if err := state.backend.Write(n); err != nil {
				return fmt.Errorf("create note: %w", err)
			}

			fmt.Fprintf(outWriter(cmd), "created %s\n", n.ID)
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Note title")
	cmd.Flags().StringVar(&typ, "type", "", "Note type: concept|argument|model|hypothesis|observation")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tags")
	cmd.Flags().StringVar(&content, "content", "", "Note body (use with --no-edit)")
	cmd.Flags().BoolVar(&noEdit, "no-edit", false, "Skip opening $EDITOR")
	cmd.Flags().StringVar(&linkTo, "link-to", "", "Immediately link to an existing note ID")
	cmd.Flags().StringVar(&annotation, "annotation", "", "Link annotation when using --link-to")
	return cmd
}
