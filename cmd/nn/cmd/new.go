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
		title       string
		typ         string
		tags        string
		content     string
		appliesWhen    string
		representation string
		expiresWhen string
		expiresStr  string
		noEdit      bool
		noSuggest   bool
		check       bool
		linkTos     []string
		linkTypes   []string
		annotations []string
		fromStdin   bool
		fromFile    string
		quick       bool
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
			if quick {
				typ = "observation"
				content = ""
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
			var expires *time.Time
			if expiresStr != "" {
				t, err := time.Parse("2006-01-02", expiresStr)
				if err != nil {
					return fmt.Errorf("--expires: invalid date %q, want YYYY-MM-DD", expiresStr)
				}
				expires = &t
			}
			n := &note.Note{
				ID:          note.GenerateID(),
				Title:       title,
				Type:        noteType,
				Status:      note.StatusDraft,
				Tags:        parsedTags,
				AppliesWhen:    appliesWhen,
				Representation: representation,
				ExpiresWhen: expiresWhen,
				Expires:     expires,
				Created:     now,
				Modified:    now,
				Body:        content,
			}

			if len(linkTos) != len(linkTypes) || len(linkTos) != len(annotations) {
				return fmt.Errorf("--link-to, --link-type, and --annotation must be paired: got %d --link-to, %d --link-type, and %d --annotation", len(linkTos), len(linkTypes), len(annotations))
			}
			for i, id := range linkTos {
				if !note.IsKnownLinkType(linkTypes[i]) {
					return fmt.Errorf("invalid --link-type %q: must be one of %s", linkTypes[i], strings.Join(note.LinkTypeOrder, ", "))
				}
				n.Links = append(n.Links, note.Link{TargetID: id, Type: linkTypes[i], Annotation: annotations[i]})
			}

			if !noEdit && isTTYFn() {
				edited, err := openEditorFn(n.Body)
				if err != nil {
					return fmt.Errorf("new: editor: %w", err)
				}
				n.Body = edited
			}

			warnIfLarge(cmd, n.Body)

			if err := state.backend.Write(n); err != nil {
				return fmt.Errorf("create note: %w", err)
			}

			w := outWriter(cmd)
			fmt.Fprintf(w, "created %s\n", n.ID)
			if check && n.Representation != "" {
				if err := runRepresentationCheck(cmd, state, n); err != nil {
					return err
				}
			}
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
	cmd.Flags().BoolVar(&check, "check", false, "Run representation graph validation after creation (requires representation field)")
	cmd.Flags().StringArrayVar(&linkTos, "link-to", nil, "Immediately link to an existing note ID (repeatable)")
	cmd.Flags().StringArrayVar(&linkTypes, "link-type", nil, "Known link type paired with --link-to (repeatable, must match count)")
	cmd.Flags().StringArrayVar(&annotations, "annotation", nil, "Link annotation paired with --link-to (repeatable, must match count)")
	cmd.Flags().StringVar(&appliesWhen, "applies-when", "", "Set applies_when field (protocol notes)")
	cmd.Flags().StringVar(&representation, "representation", "", "Set representation type (ontology|taxonomy|axiom)")
	cmd.Flags().StringVar(&expiresWhen, "expires-when", "", "Set conditional expiration (plain text condition, e.g. 'when the PR is merged')")
	cmd.Flags().StringVar(&expiresStr, "expires", "", "Set expiration date (YYYY-MM-DD); note appears in nn list --expired after this date")
	cmd.Flags().BoolVar(&fromStdin, "from-stdin", false, "Read note body from stdin")
	cmd.Flags().StringVar(&fromFile, "from-file", "", "Scaffold note body from ast outline of a source file")
	cmd.Flags().BoolVar(&quick, "quick", false, "Quick capture: sets type=observation, status=draft, content empty")
	return cmd
}
