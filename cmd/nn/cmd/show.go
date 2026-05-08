package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

// virtualProtocol is a hardcoded protocol always included in nn show --global output.
// Body contains the full constraint text; AppliesWhen is a one-line application rule
// shown in compact --global output so the LLM can decide whether to fetch the full body.
type virtualProtocol struct {
	ID          string
	Title       string
	AppliesWhen string
	Body        string
}

// virtualGlobalProtocols are hardcoded protocols always included in nn show --global output.
// Add entries here to register additional tool-level meta-protocols.
var virtualGlobalProtocols = []virtualProtocol{
	{
		ID:          "virtual-nn-capture-discipline",
		Title:       "Protocol: nn-capture-discipline",
		AppliesWhen: "before any action that reads from a source not authored this session",
		Body: "Before running the search, write: 'Search rationale: [action] requires knowing [X].' " +
			"The search query must contain at least one word from X.\n\n" +
			"Every action requires a preceding `nn list --search \"<topic>\" --show-first --json` result in the transcript, " +
			"except actions on the allow-list below. " +
			"The search result must appear as the immediately preceding tool result before the action. " +
			"A search result satisfies the gate only if the search query contains at least one word from the stated search rationale — " +
			"a search result whose query shares no word with the stated search rationale does not satisfy this gate. " +
			"After reviewing the results, cite at least one result title: either the title that covers the question, or a title from the results and explain why it does not cover the stated search rationale.\n\n" +
			"**Allow-list (no gate required):**\n" +
			"- Running, editing, or reading a file that appears in a prior Write, Edit, or Bash tool call in this session\n" +
			"- Running a command that produces output solely from local code or state present in this session (e.g. tests, builds, linters)\n" +
			"- Fetching output from an execution system you triggered or are operating in this session (e.g. CI run you initiated, container you started), where the result did not exist before this session\n\n" +
			"Everything else — web search, URL fetch, reading documentation, spawning an agent to gather facts, " +
			"reading memory files, reading any file not on the allow-list — requires the gate.\n\n" +
			"After the action completes, quote a verbatim excerpt from the result (a string literally present in the tool output). " +
			"If the quoted excerpt is `[]`, write \"zero results returned\" and either capture or skip. " +
			"Otherwise, cite at least one result title from the quoted excerpt: either the title that covers the question, " +
			"or a title from the results and an explanation of why it does not cover the stated search rationale. " +
			"Then either capture the finding with `nn new` / `nn update` / `nn link`, " +
			"or skip with the verbatim excerpt, the source, " +
			"and the statement \"result is a runtime value\" when the result is execution output with no reuse across sessions.\n",
	},
	{
		ID:          "virtual-nn-unexpected-failure",
		Title:       "Protocol: search nn before workarounding an unexpected command failure",
		AppliesWhen: "when a Bash command, CLI tool, or test fails in an unexpected or non-obvious way",
		Body: "When a command fails in a surprising way — non-obvious error message, tool behaves differently than expected, " +
			"test fails for unknown reasons — run `nn list --search \"<topic>\"` before attempting a workaround or fix. " +
			"Prior sessions may have captured the root cause, a known workaround, or a relevant constraint. " +
			"Acting without searching risks repeating a diagnosed problem.\n",
	},
}

// protocolDerivationBlock is appended to every protocol note in plain-text display.
// It instructs any LLM reading the note to derive how the protocol applies before responding.
const protocolDerivationBlock = `
<!-- nn:display-only — the following block is injected by nn show and is NOT part of the note body. Do not include it in nn update --content or any note edit. -->

## Protocols

For each protocol listed above:
1. Evaluate its applies_when condition against the current request.
2. If it holds: run ` + "`nn show <id>`" + ` to load the full body, then treat it as a binding constraint.
3. If a protocol has no applies_when field: add an applies_when value via ` + "`nn update <id> --field applies_when --value \"...\"`" + ` before proceeding.

Before responding to any message this session, output a ` + "`## Protocols`" + ` block.
For each active protocol, write one sentence stating how it applies to this specific
request — or "not applicable" if it doesn't. Place this block before your substantive
response.
`

func newShowCmd(state *rootState) *cobra.Command {
	var linkedFrom string
	var jsonOut bool
	var depth int
	var global bool

	cmd := &cobra.Command{
		Use:   "show <id-or-title> [<id-or-title>...]",
		Short: "Print note content to stdout (accepts ID or title substring; --depth N for graph traversal)",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			w := outWriter(cmd)

			if global {
				all, err := state.backend.List()
				if err != nil {
					return fmt.Errorf("show --global: %w", err)
				}
				first := true
				for _, vp := range virtualGlobalProtocols {
					if !first {
						fmt.Fprintln(w, "---")
					}
					first = false
					fmt.Fprintf(w, "id: %s\ntitle: %s\ntype: protocol\nstatus: permanent\napplies_when: %s\n---\n\n%s", vp.ID, vp.Title, vp.AppliesWhen, vp.Body)
				}
				for _, n := range all {
					if n.Type != note.TypeProtocol {
						continue
					}
					hasGoverns := false
					for _, lnk := range n.Links {
						if lnk.Type == "governs" {
							hasGoverns = true
							break
						}
					}
					if hasGoverns {
						continue
					}
					if !first {
						fmt.Fprintln(w, "---")
					}
					first = false
					fmt.Fprintf(w, "id: %s\ntitle: %s\n", n.ID, n.Title)
					if n.AppliesWhen != "" {
						fmt.Fprintf(w, "applies_when: %s\n", n.AppliesWhen)
					}
				}
				fmt.Fprint(w, protocolDerivationBlock)
				return nil
			}

			if depth > 0 {
				if len(args) != 1 {
					return fmt.Errorf("show --depth: provide exactly one ID")
				}
				root, err := resolveNote(state, args[0])
				if err != nil {
					return fmt.Errorf("show --depth: %w", err)
				}
				all, err := state.backend.List()
				if err != nil {
					return fmt.Errorf("show --depth: list: %w", err)
				}
				byID := make(map[string]*note.Note, len(all))
				for _, n := range all {
					byID[n.ID] = n
				}
				entries := bfsDepth(root, byID, depth)
				if jsonOut {
					return printDepthJSON(w, entries)
				}
				return printDepthMarkdown(w, entries)
			}

			if linkedFrom != "" {
				src, err := resolveNote(state, linkedFrom)
				if err != nil {
					return fmt.Errorf("show --linked-from: %w", err)
				}
				all, err := state.backend.List()
				if err != nil {
					return fmt.Errorf("show --linked-from: list: %w", err)
				}
				for i, lnk := range src.Links {
					n, err := state.backend.Read(lnk.TargetID)
					if err != nil {
						continue // skip broken links silently
					}
					if i > 0 {
						fmt.Fprintln(w, "---")
					}
					protos := findGoverningProtocols(n.ID, all)
					if len(protos) > 0 {
						fmt.Fprintf(w, "governing protocols:\n")
						for _, p := range protos {
							fmt.Fprintf(w, "  - [%s] %s\n", p.ID, p.Title)
						}
						fmt.Fprintln(w)
					}
					data, err := n.Marshal()
					if err != nil {
						return fmt.Errorf("show: marshal: %w", err)
					}
					fmt.Fprint(w, string(data))
				}
				return nil
			}

			if len(args) == 0 {
				return fmt.Errorf("show: provide at least one ID or use --linked-from")
			}

			all, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("show: list: %w", err)
			}

			for i, query := range args {
				if i > 0 {
					fmt.Fprintln(w, "---")
				}
				if vp := findVirtualProtocol(query); vp != nil {
					fmt.Fprintf(w, "---\nid: %s\ntitle: %s\napplies_when: %s\ntype: protocol\nstatus: permanent\n---\n\n%s", vp.ID, vp.Title, vp.AppliesWhen, vp.Body)
					continue
				}
				n, err := resolveNote(state, query)
				if err != nil {
					return fmt.Errorf("show: %w", err)
				}
				appendAccessLog(n.ID)
				protos := findGoverningProtocols(n.ID, all)

				if jsonOut {
					type protoRef struct {
						ID    string `json:"id"`
						Title string `json:"title"`
					}
					type showJSON struct {
						ID                  string     `json:"id"`
						Title               string     `json:"title"`
						Type                string     `json:"type"`
						Status              string     `json:"status"`
						Tags                []string   `json:"tags"`
						Created             string     `json:"created"`
						Modified            string     `json:"modified"`
						Body                string     `json:"body"`
						GoverningProtocols  []protoRef `json:"governing_protocols"`
					}
					refs := make([]protoRef, len(protos))
					for j, p := range protos {
						refs[j] = protoRef{ID: p.ID, Title: p.Title}
					}
					if refs == nil {
						refs = []protoRef{}
					}
					out := showJSON{
						ID:                 n.ID,
						Title:              n.Title,
						Type:               string(n.Type),
						Status:             string(n.Status),
						Tags:               n.Tags,
						Created:            n.Created.UTC().Format("2006-01-02T15:04:05Z"),
						Modified:           n.Modified.UTC().Format("2006-01-02T15:04:05Z"),
						Body:               n.Body,
						GoverningProtocols: refs,
					}
					if out.Tags == nil {
						out.Tags = []string{}
					}
					enc := json.NewEncoder(w)
					enc.SetIndent("", "  ")
					if err := enc.Encode(out); err != nil {
						return fmt.Errorf("show: json: %w", err)
					}
					continue
				}

				if len(protos) > 0 {
					fmt.Fprintf(w, "governing protocols:\n")
					for _, p := range protos {
						fmt.Fprintf(w, "  - [%s] %s\n", p.ID, p.Title)
					}
					fmt.Fprintln(w)
				}
				data, err := n.Marshal()
				if err != nil {
					return fmt.Errorf("show: marshal: %w", err)
				}
				fmt.Fprint(w, string(data))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&linkedFrom, "linked-from", "", "Show all notes linked from this ID")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output note as JSON with governing_protocols")
	cmd.Flags().IntVar(&depth, "depth", 0, "Traverse outgoing links to this depth and print all reachable notes")
	cmd.Flags().BoolVar(&global, "global", false, "Show all global protocol notes (type:protocol with no outgoing governs links)")
	return cmd
}

// appendAccessLog records a note retrieval to the advisory access log.
// Failures are silently ignored — the log is advisory only.
func appendAccessLog(id string) {
	cfgDir := os.Getenv("NN_CONFIG_DIR")
	if cfgDir == "" {
		xdg := os.Getenv("XDG_CONFIG_HOME")
		if xdg == "" {
			home, _ := os.UserHomeDir()
			xdg = filepath.Join(home, ".config")
		}
		cfgDir = filepath.Join(xdg, "nn")
	}
	_ = os.MkdirAll(cfgDir, 0o755)
	f, err := os.OpenFile(filepath.Join(cfgDir, "access.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s show %s\n", time.Now().UTC().Format(time.RFC3339), id)
}

// findGoverningProtocols returns all notes that link to targetID with type "governs".
func findGoverningProtocols(targetID string, all []*note.Note) []*note.Note {
	var result []*note.Note
	for _, n := range all {
		for _, lnk := range n.Links {
			if lnk.TargetID == targetID && lnk.Type == "governs" {
				result = append(result, n)
				break
			}
		}
	}
	return result
}

// resolveNote finds a note by exact ID or case-insensitive title substring.
func resolveNote(state *rootState, query string) (*note.Note, error) {
	n, err := state.backend.Read(query)
	if err == nil {
		return n, nil
	}
	all, listErr := state.backend.List()
	if listErr != nil {
		return nil, fmt.Errorf("%w", err)
	}
	type match struct{ id, title string }
	var matches []match
	for _, candidate := range all {
		if strings.Contains(strings.ToLower(candidate.Title), strings.ToLower(query)) {
			matches = append(matches, match{candidate.ID, candidate.Title})
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no note found matching %q", query)
	case 1:
		return state.backend.Read(matches[0].id)
	default:
		return nil, fmt.Errorf("ambiguous query %q — %d matches; use full ID", query, len(matches))
	}
}

// findVirtualProtocol returns the virtualProtocol whose ID matches query exactly,
// or nil if no virtual protocol matches.
func findVirtualProtocol(query string) *virtualProtocol {
	for i, vp := range virtualGlobalProtocols {
		if vp.ID == query {
			return &virtualGlobalProtocols[i]
		}
	}
	return nil
}
