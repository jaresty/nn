package cmd

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/ast"
	"github.com/jaresty/nn/internal/note"
)

func newNewCmd(state *rootState) *cobra.Command {
	var (
		title      string
		typ        string
		tags       string
		content    string
		noEdit     bool
		noSuggest  bool
		linkTo     string
		annotation string
		fromStdin  bool
		fromFile   string
	)

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create a new note",
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromStdin {
				data, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return fmt.Errorf("new: read stdin: %w", err)
				}
				if content == "" {
					content = string(data)
				}
			}
			if fromFile != "" {
				f, err := ast.Parse(fromFile)
				if err != nil {
					return fmt.Errorf("new: --from-file: %w", err)
				}
				if title == "" {
					title = filepath.Base(fromFile)
				}
				if content == "" {
					var sb strings.Builder
					sb.WriteString("file: ")
					sb.WriteString(fromFile)
					sb.WriteString("  language: ")
					sb.WriteString(f.Language)
					sb.WriteString("\n\n")
					for _, sym := range f.Symbols {
						if sym.Kind == "import" {
							sb.WriteString("imports: ")
							sb.WriteString(sym.Name)
							sb.WriteString("\n")
							continue
						}
						sb.WriteString(sym.Signature)
						sb.WriteString("\n")
					}
					content = sb.String()
				}
			}
			if typ == "" {
				return fmt.Errorf("--type is required (concept|argument|model|hypothesis|observation|question|protocol)")
			}
			noteType := note.Type(typ)
			if !noteType.IsValid() {
				return fmt.Errorf("invalid --type %q: must be concept|argument|model|hypothesis|observation|question|protocol", typ)
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

			warnIfLarge(cmd, n.Body)

			if err := state.backend.Write(n); err != nil {
				return fmt.Errorf("create note: %w", err)
			}

			w := outWriter(cmd)
			fmt.Fprintf(w, "created %s\n", n.ID)
			if !noSuggest {
				printSuggestions(w, state, n)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "Note title")
	cmd.Flags().StringVar(&typ, "type", "", "Note type: concept|argument|model|hypothesis|observation|question|protocol")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma-separated tags")
	cmd.Flags().StringVar(&content, "content", "", "Note body (use with --no-edit)")
	cmd.Flags().BoolVar(&noEdit, "no-edit", false, "Skip opening $EDITOR")
	cmd.Flags().BoolVar(&noSuggest, "no-suggest", false, "Suppress post-write link and tag suggestions")
	cmd.Flags().StringVar(&linkTo, "link-to", "", "Immediately link to an existing note ID")
	cmd.Flags().StringVar(&annotation, "annotation", "", "Link annotation when using --link-to")
	cmd.Flags().BoolVar(&fromStdin, "from-stdin", false, "Read note body from stdin")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "Scaffold note body from ast outline of a source file")
	return cmd
}
