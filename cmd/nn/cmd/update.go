package cmd

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

var updateNowFn = func() time.Time { return time.Now().In(time.Local) }

func newUpdateCmd(state *rootState) *cobra.Command {
	var (
		title          string
		tags           string
		tagsAdd        []string
		tagsRemove     []string
		content        string
		appendS        string
		typ            string
		status         string
		appliesWhen    string
		expiresWhen    string
		expiresStr     string
		sinceStr       string
		fromStdin      bool
		replaceSection string
		noEdit         bool
	)

	cmd := &cobra.Command{
		Use:   "update <id-or-title>",
		Short: "Update an existing note's title, type, tags, status, or body",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if content != "" && appendS != "" {
				return fmt.Errorf("--content and --append are mutually exclusive")
			}
			if fromStdin && content != "" {
				return fmt.Errorf("--stdin and --content are mutually exclusive")
			}
			if replaceSection != "" && content == "" && !fromStdin {
				return fmt.Errorf("--replace-section requires --content or --stdin")
			}
			willOpenEditor := !noEdit && isTTYFn()
			if title == "" && tags == "" && content == "" && appendS == "" &&
				typ == "" && status == "" && appliesWhen == "" && expiresWhen == "" && expiresStr == "" && !fromStdin && replaceSection == "" &&
				len(tagsAdd) == 0 && len(tagsRemove) == 0 && !willOpenEditor {
				return fmt.Errorf("at least one of --title, --tags, --tags-add, --tags-remove, --content, --stdin, --append, --type, --status, --applies-when, --expires-when, --expires, --replace-section is required")
			}
			isDailyAlias := strings.ToLower(args[0]) == "daily"
			if sinceStr == "" && !isDailyAlias {
				return fmt.Errorf("--since is required: read 'modified:' from nn show output and pass it to prevent overwriting concurrent changes")
			}

			n, err := resolveNote(state, args[0])
			if err != nil {
				return fmt.Errorf("update: %w", err)
			}

			if sinceStr != "" {
				since, parseErr := time.Parse(time.RFC3339, sinceStr)
				if parseErr != nil {
					return fmt.Errorf("--since: invalid timestamp %q, want RFC3339", sinceStr)
				}
				if n.Modified.Sub(since).Abs() > time.Second {
					return fmt.Errorf("note was modified since %s; re-read and retry", sinceStr)
				}
			}

			if fromStdin {
				data, readErr := io.ReadAll(cmd.InOrStdin())
				if readErr != nil {
					return fmt.Errorf("update: read stdin: %w", readErr)
				}
				content = stripLinksSection(string(data))
			}

			if title != "" {
				n.Title = title
			}
			if tags != "" {
				var parsed []string
				for _, t := range strings.Split(tags, ",") {
					if t = strings.TrimSpace(t); t != "" {
						parsed = append(parsed, t)
					}
				}
				n.Tags = parsed
			}
			if len(tagsAdd) > 0 || len(tagsRemove) > 0 {
				existing := make(map[string]struct{}, len(n.Tags))
				for _, t := range n.Tags {
					existing[t] = struct{}{}
				}
				for _, t := range tagsAdd {
					existing[t] = struct{}{}
				}
				for _, t := range tagsRemove {
					delete(existing, t)
				}
				merged := make([]string, 0, len(existing))
				for t := range existing {
					merged = append(merged, t)
				}
				n.Tags = merged
			}
			if typ != "" {
				t := note.Type(typ)
				if !t.IsValid() {
					return fmt.Errorf("invalid type %q: must be one of %s", typ, strings.Join(note.ValidTypes(), ", "))
				}
				n.Type = t
			}
			if status != "" {
				s := note.Status(status)
				if !s.IsValid() {
					return fmt.Errorf("invalid status %q: must be draft, reviewed, or permanent", status)
				}
				n.Status = s
			}
			if appliesWhen != "" {
				n.AppliesWhen = appliesWhen
			}
			if expiresWhen != "" {
				n.ExpiresWhen = expiresWhen
			}
			if expiresStr != "" {
				t, err := time.Parse("2006-01-02", expiresStr)
				if err != nil {
					return fmt.Errorf("--expires: invalid date %q, want YYYY-MM-DD", expiresStr)
				}
				n.Expires = &t
			}
			if willOpenEditor && content == "" && appendS == "" && replaceSection == "" && !fromStdin {
				edited, err := openEditorFn(strings.TrimSpace(n.Body))
				if err != nil {
					return fmt.Errorf("update: editor: %w", err)
				}
				n.Body = edited
			} else if replaceSection != "" {
				replaced, replErr := replaceMarkdownSection(n.Body, replaceSection, content)
				if replErr != nil {
					return fmt.Errorf("update: %w", replErr)
				}
				n.Body = replaced
			} else if content != "" {
				n.Body = stripLinksSection(content)
			} else if appendS != "" {
				if n.Body == "" {
					n.Body = appendS
				} else {
					n.Body = n.Body + "\n\n" + appendS
				}
			}
			n.Modified = updateNowFn()
			warnIfLarge(cmd, n.Body)

			if err := state.backend.Update(n); err != nil {
				return fmt.Errorf("update: %w", err)
			}
			fmt.Fprintf(outWriter(cmd), "updated %s\nmodified: %s\n", n.ID, n.Modified.Format(time.RFC3339))
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "New note title")
	cmd.Flags().StringVar(&tags, "tags", "", "Replace tags (comma-separated)")
	cmd.Flags().StringArrayVar(&tagsAdd, "tags-add", nil, "Add a tag (repeatable)")
	cmd.Flags().StringArrayVar(&tagsRemove, "tags-remove", nil, "Remove a tag (repeatable)")
	cmd.Flags().StringVar(&content, "content", "", "Replace note body entirely")
	cmd.Flags().StringVar(&appendS, "append", "", "Append text to note body")
	cmd.Flags().StringVar(&typ, "type", "", "Change note type")
	cmd.Flags().StringVar(&status, "status", "", "Set note status (draft|reviewed|permanent)")
	cmd.Flags().StringVar(&appliesWhen, "applies-when", "", "Set applies_when field (protocol notes)")
	cmd.Flags().StringVar(&expiresWhen, "expires-when", "", "Set conditional expiration (plain text condition)")
	cmd.Flags().StringVar(&expiresStr, "expires", "", "Set expiration date (YYYY-MM-DD)")
	cmd.Flags().BoolVar(&fromStdin, "stdin", false, "Read note body from stdin")
	cmd.Flags().StringVar(&replaceSection, "replace-section", "", "Replace named level-2 section (case-insensitive)")
	cmd.Flags().BoolVar(&noEdit, "no-edit", false, "Skip opening $EDITOR")
	cmd.Flags().StringVar(&sinceStr, "since", "", "Reject update if note was modified after this RFC3339 timestamp")
	return cmd
}

// stripLinksSection removes any "## Links" section from user-provided body content.
// The Links section is owned by the note graph (nn link/unlink); passing it
// through --content or --stdin would cause Marshal to write it twice, creating
// duplicate graph edges on the next parse.
func stripLinksSection(body string) string {
	const marker = "\n## Links\n"
	if idx := strings.Index(body, marker); idx != -1 {
		return strings.TrimRight(body[:idx], "\n")
	}
	return body
}

// replaceMarkdownSection replaces the content of a level-2 heading section
// (case-insensitive match) with newContent. Returns an error if not found.
func replaceMarkdownSection(body, heading, newContent string) (string, error) {
	lines := strings.Split(body, "\n")
	targetHeading := strings.ToLower(strings.TrimSpace(heading))

	startIdx := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			sectionTitle := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(trimmed, "## ")))
			if sectionTitle == targetHeading {
				startIdx = i
				break
			}
		}
	}
	if startIdx == -1 {
		var appended strings.Builder
		appended.WriteString(body)
		if body != "" && !strings.HasSuffix(body, "\n") {
			appended.WriteString("\n")
		}
		appended.WriteString("\n## ")
		appended.WriteString(heading)
		appended.WriteString("\n\n")
		appended.WriteString(newContent)
		return appended.String(), nil
	}

	// Find where the section ends (next level-2 heading or end of body).
	endIdx := len(lines)
	for i := startIdx + 1; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "## ") {
			endIdx = i
			break
		}
	}

	// Rebuild: keep heading, replace content, keep rest.
	var result []string
	result = append(result, lines[:startIdx+1]...)
	result = append(result, "")
	result = append(result, newContent)
	if endIdx < len(lines) {
		result = append(result, "")
		result = append(result, lines[endIdx:]...)
	}
	return strings.Join(result, "\n"), nil
}
