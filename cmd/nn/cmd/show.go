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

// nowFn is the time source for daily note date labels. Tests override it to simulate timezone offsets.
var nowFn = time.Now

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
		AppliesWhen: "before any Read tool call or file-reading Bash tool call",
		Body: "Prior knowledge of a topic is not an exemption from this gate — it is the reason the search is most likely to return a relevant note. " +
			"Skip resistance is a signal the search is especially needed.\n\n" +
			"Before every Read tool call or file-reading Bash tool call, the immediately preceding assistant text block must contain exactly one of: " +
			"`Gate: allow-listed — [specific allow-list item]` if the action qualifies for an allow-list exemption, " +
			"or `Gate: Search rationale: [action] requires knowing [X]` if the gate applies. " +
			"An evaluator determines compliance by locating the tool call in the transcript, reading backward to the first assistant text block, " +
			"and checking for the literal string `Gate:` within that block. " +
			"A `Gate:` line present in any earlier assistant text block, in any later block, or embedded within the tool call itself does not satisfy this clause. " +
			"The search query must contain at least one word from the topic named after `Search rationale:`.\n\n" +
			"Every Read tool call or file-reading Bash tool call requires a preceding `nn list --search \"<topic>\" --json` result in the transcript " +
			"(no `|` in the same tool call — a tool call containing `nn list --search` followed by `|` is non-compliant; use `--limit N` to restrict result count instead), " +
			"except actions on the allow-list below. " +
			"The `nn list --search` tool call must be the tool call whose result block occupies the position immediately before the gated tool call in the ordered sequence of tool call result blocks. " +
			"A tool call result block is defined as one complete response returned by a single tool invocation, delimited by the tool invocation boundary in the transcript. " +
			"Zero tool call result blocks of any kind may intervene between the `nn list --search` result block and the gated tool call. " +
			"An evaluator determines compliance by counting backward from the gated tool call: the first tool call result block encountered must be the `nn list --search` result. " +
			"A search result satisfies the gate only if the search query contains at least one word from the stated search rationale — " +
			"a search result whose query shares no word with the stated search rationale does not satisfy this gate. " +
			"The assistant text block immediately following the `nn list --search` result block must contain a verbatim substring that exactly matches the value of at least one `title` field in the returned JSON array — " +
			"an evaluator determines compliance by parsing the result as JSON, extracting all `title` field values, and checking whether any appears verbatim in the subsequent assistant text block. " +
			"If the returned array is `[]`, the assistant must issue a second `nn list --search` tool call with a rephrased query before proceeding — a single zero-result search does not satisfy the gate. " +
			"The assistant text block after the second search must contain the literal string `zero results returned` if the second search also returns `[]`. " +
			"If the JSON array contains any object whose `title` field value shares at least one word with the `Search rationale:` string, " +
			"the assistant must issue an `nn show <id>` tool call where `<id>` is the `id` field value of the first object in the array (array index 0) whose `title` field satisfies the word-sharing condition — " +
			"an evaluator determines compliance by identifying the lowest-index matching object, extracting its `id`, " +
			"and checking whether a subsequent `nn show <id>` tool call containing that exact `id` appears in the transcript before the gated action.\n\n" +
			"**Allow-list (no gate required):**\n" +
			"- Running, editing, or reading a file that is the target path of a prior Write or Edit tool call in this session — " +
			"a Bash tool call satisfies this only if its text contains `>`, `>>`, `tee`, `cp`, `mv`, or `install` targeting that path; " +
			"a Bash call that only reads or lists the file (e.g. `find`, `ls`, `grep`, `cat`) does not satisfy this condition\n" +
			"- Running a command that produces output solely from local code or state present in this session (e.g. tests, builds, linters)\n" +
			"- Fetching output from an execution system you triggered or are operating in this session (e.g. CI run you initiated, container you started), where the result did not exist before this session\n" +
			"- Fetching live operational state from a system where: (a) a specific resource identifier (branch, PR number, host, job ID) for that system appears in the conversation above this action, and (b) the fetched result is machine-generated output (JSON, status code, log line) rather than human-authored content\n\n" +
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
		ID:          "virtual-nn-cli-reference",
		Title:       "Protocol: nn CLI reference",
		AppliesWhen: "always — reference for valid nn command flags, types, and statuses",
		Body: "**nn new** `--title \"...\" --type <type> --content \"...\" --no-edit [--tags \"t1,t2\"] [--link-to <id> --annotation \"...\"] [--applies-when \"...\"] [--expires YYYY-MM-DD] [--expires-when \"condition\"]`\n" +
			"Valid --type: concept|argument|model|hypothesis|observation|question|protocol\n" +
			"New notes are always created as draft. Promote with: `nn update <id> --status reviewed`\n\n" +
			"**nn update** `<id> --since <RFC3339> --status <status>` | `--title \"...\"` | `--applies-when \"...\"` | `--expires YYYY-MM-DD` | `--expires-when \"condition\"` | `--content \"...\" --no-edit`\n" +
			"Valid --status: draft|reviewed|permanent\n" +
			"--since is required: read 'modified:' from nn show output; update is rejected if the note was changed after that timestamp\n\n" +
			"**nn remind** `\"content\" [--for N] [--expires YYYY-MM-DD]` — creates observation tagged 'reminder', permanent, expires today+1d by default; surfaces in nn show --global\n\n" +
			"**nn list** `--search \"<q>\" --show-first --json` | `--type <type>` | `--status <status>` | `--orphan` | `--since <ISO>` | `--expired` | `--has-expires`\n\n" +
			"**nn show** `<id>` | `--global`\n\n" +
			"**nn link** `<from> <to> --type <type> --annotation \"...\"`\n" +
			"Valid --type: refines|contradicts|source-of|extends|supports|questions|governs\n\n" +
			"**nn show** `<id> --depth N` — traverse N hops of outgoing links from a note\n\n" +
			"**nn path** `<id-a> <id-b>` — shortest path between two notes\n\n" +
			"**nn clusters** — topological clusters via label propagation\n\n" +
			"**nn list** `--similar <id>` — BM25 similarity (notes sharing vocabulary but not linked)\n\n" +
			"**nn graph** `[--json]` — export full graph as JSON `{ \"nodes\": [...], \"edges\": [...] }`\n\n" +
			"**nn graph show** `--focus <id> [--depth N]` — subgraph centered on a note (LLM-facing; default depth 2)\n\n" +
			"For the full command reference, invoke `/nn-guide`.\n",
	},
	{
		ID:          "virtual-nn-error-handling",
		Title:       "Protocol: search nn before workarounding an unexpected command failure",
		AppliesWhen: "when a Bash command, CLI tool, or test fails",
		Body: "When a command fails, run `nn list --search \"<topic>\" --json` before attempting a workaround or fix. " +
			"Prior sessions may have captured the root cause, a known workaround, or a relevant constraint. " +
			"Acting without searching risks repeating a diagnosed problem.\n\n" +
			"**Skip condition A — pre-declared:** if the immediately preceding assistant turn contains a line beginning " +
			"`Expected FAIL: <reason>` (case-insensitive), the lookup is not required. " +
			"Write: `Skipping lookup: cited \"Expected FAIL: <reason>\"` quoting the declaration verbatim.\n\n" +
			"**Skip condition B — self-evident:** if the error output contains a substring that names the cause without ambiguity " +
			"— such as \"No such file\", \"command not found\", \"unrecognized token\", or \"exit status 1\" from a grep/find/cat invocation — " +
			"the lookup is not required. " +
			"Write: `Skipping lookup: self-evident — \"<quoted error substring>\"` quoting the exact substring from the tool output verbatim. " +
			"Absence of either skip declaration means the lookup is required.\n",
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
				now := time.Now().UTC()
				var activeReminders []*note.Note
				for _, n := range all {
					hasReminder := false
					for _, tag := range n.Tags {
						if tag == "reminder" {
							hasReminder = true
							break
						}
					}
					if hasReminder && (n.Expires == nil || !n.Expires.Before(now)) {
						activeReminders = append(activeReminders, n)
					}
				}
				if len(activeReminders) > 0 {
					fmt.Fprintln(w, "---")
					fmt.Fprintln(w, "## Reminders")
					fmt.Fprintln(w)
					for _, n := range activeReminders {
						fmt.Fprintf(w, "### %s\n\n", n.Title)
						if n.Expires != nil {
							fmt.Fprintf(w, "expires: %s\n", n.Expires.Format("2006-01-02"))
						}
						if n.ExpiresWhen != "" {
							fmt.Fprintf(w, "expires_when: %s\n", n.ExpiresWhen)
						}
						fmt.Fprintf(w, "\n%s\n\n", n.Body)
					}
				}
				if dn, dnErr := resolveDailyNote(state); dnErr == nil {
					fmt.Fprintln(w, "---")
					byID := make(map[string]*note.Note, len(all))
					for _, a := range all {
						byID[a.ID] = a
					}
					backlinkers := findBacklinkers(dn.ID, all)
					if data, merr := dn.Marshal(); merr == nil {
						rendered := renderWithResolvedLinks(string(data), dn, byID, backlinkers)
						fmt.Fprint(w, rendered)
					} else {
						fmt.Fprintf(w, "id: %s\ntitle: %s\ntype: observation\nstatus: %s\ntags: daily\n---\n\n%s\n", dn.ID, dn.Title, dn.Status, dn.Body)
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
				byID := make(map[string]*note.Note, len(all))
				for _, a := range all {
					byID[a.ID] = a
				}
				data, err := n.Marshal()
				if err != nil {
					return fmt.Errorf("show: marshal: %w", err)
				}
				backlinkers := findBacklinkers(n.ID, all)
				rendered := renderWithResolvedLinks(string(data), n, byID, backlinkers)
				rendered = injectFreshness(rendered, n.Modified)
				fmt.Fprint(w, rendered)
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
// The special query "daily" resolves to today's Daily: YYYY-MM-DD note, creating it if absent.
func resolveNote(state *rootState, query string) (*note.Note, error) {
	if strings.ToLower(query) == "daily" {
		return resolveDailyNote(state)
	}
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

// resolveDailyNote finds or creates today's Daily: YYYY-MM-DD note.
// Created notes are tagged "daily" and expire in 7 days.
func resolveDailyNote(state *rootState) (*note.Note, error) {
	today := nowFn().Format("2006-01-02")
	todayTitle := "Daily: " + today

	all, err := state.backend.List()
	if err != nil {
		return nil, fmt.Errorf("daily: list: %w", err)
	}
	for _, candidate := range all {
		if candidate.Title == todayTitle {
			return state.backend.Read(candidate.ID)
		}
	}

	yesterday := nowFn().AddDate(0, 0, -1).Format("2006-01-02")
	yesterdayTitle := "Daily: " + yesterday
	var body string
	for _, candidate := range all {
		if candidate.Title == yesterdayTitle {
			yn, readErr := state.backend.Read(candidate.ID)
			if readErr == nil && yn.Body != "" {
				body = "### Yesterday\n\n" + yn.Body
			}
			break
		}
	}

	expires := time.Now().UTC().AddDate(0, 0, 7)
	now := time.Now().UTC()
	n := &note.Note{
		ID:       note.GenerateID(),
		Title:    todayTitle,
		Type:     note.TypeObservation,
		Status:   note.StatusDraft,
		Tags:     []string{"daily"},
		Expires:  &expires,
		Created:  now,
		Modified: now,
		Body:     body,
	}
	if err := state.backend.Write(n); err != nil {
		return nil, fmt.Errorf("daily: create: %w", err)
	}
	return n, nil
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

// findBacklinkers returns all notes in all that link to targetID (any link type).
func findBacklinkers(targetID string, all []*note.Note) []*note.Note {
	var result []*note.Note
	for _, n := range all {
		for _, lnk := range n.Links {
			if lnk.TargetID == targetID {
				result = append(result, n)
				break
			}
		}
	}
	return result
}

// injectFreshness inserts a "freshness: <tier>" line after the closing "---" of the
// YAML frontmatter in a marshaled note string.
func injectFreshness(raw string, modified time.Time) string {
	const sep = "---\n"
	// Find the second "---\n" which closes the frontmatter.
	first := strings.Index(raw, sep)
	if first < 0 {
		return raw
	}
	second := strings.Index(raw[first+len(sep):], sep)
	if second < 0 {
		return raw
	}
	insertAt := first + len(sep) + second + len(sep)
	label := freshnessLabel(modified)
	return raw[:insertAt] + "freshness: " + label + "\n" + raw[insertAt:]
}

// freshnessLabel returns a full staleness description including tier, human-readable age, and action hint.
// Tiers: fresh (< 3 days), aging (3–14 days), stale (> 14 days).
func freshnessLabel(modified time.Time) string {
	age := time.Since(modified)
	ageStr := humanAge(age)
	switch {
	case age < 3*24*time.Hour:
		return "fresh (" + ageStr + " — likely current)"
	case age < 14*24*time.Hour:
		return "aging (" + ageStr + " — may need recheck)"
	default:
		return "stale (" + ageStr + " — content may be outdated, verify before use)"
	}
}

// humanAge formats a duration as a short human-readable age string like "2 days ago" or "5 hours ago".
func humanAge(d time.Duration) string {
	switch {
	case d < time.Hour:
		m := int(d.Minutes())
		if m <= 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

// renderWithResolvedLinks replaces the raw ## Links section in marshaled note output
// with title-resolved link lines, and appends a ## Backlinks section.
// Target IDs are resolved via byID; unresolved IDs fall back to the bare ID.
func renderWithResolvedLinks(raw string, n *note.Note, byID map[string]*note.Note, backlinkers []*note.Note) string {
	var buf strings.Builder
	if len(n.Links) > 0 {
		const linkSection = "\n## Links\n\n"
		cut := strings.Index(raw, linkSection)
		if cut >= 0 {
			buf.WriteString(raw[:cut])
			buf.WriteString(linkSection)
			for _, lnk := range n.Links {
				title := lnk.TargetID
				if t, ok := byID[lnk.TargetID]; ok {
					title = t.Title
				}
				switch {
				case lnk.Type != "" && lnk.Status != "":
					fmt.Fprintf(&buf, "- [[%s|%s]] [%s] {%s} — %s\n", lnk.TargetID, title, lnk.Type, lnk.Status, lnk.Annotation)
				case lnk.Type != "":
					fmt.Fprintf(&buf, "- [[%s|%s]] [%s] — %s\n", lnk.TargetID, title, lnk.Type, lnk.Annotation)
				case lnk.Status != "":
					fmt.Fprintf(&buf, "- [[%s|%s]] {%s} — %s\n", lnk.TargetID, title, lnk.Status, lnk.Annotation)
				default:
					fmt.Fprintf(&buf, "- [[%s|%s]] — %s\n", lnk.TargetID, title, lnk.Annotation)
				}
			}
		} else {
			buf.WriteString(raw)
		}
	} else {
		buf.WriteString(raw)
	}
	if len(backlinkers) > 0 {
		fmt.Fprintf(&buf, "\n## Backlinks (%d)\n\n", len(backlinkers))
		for _, b := range backlinkers {
			for _, lnk := range b.Links {
				if lnk.TargetID == n.ID {
					if lnk.Annotation != "" {
						fmt.Fprintf(&buf, "- [[%s|%s]] — %s\n", b.ID, b.Title, lnk.Annotation)
					} else {
						fmt.Fprintf(&buf, "- [[%s|%s]]\n", b.ID, b.Title)
					}
					break
				}
			}
		}
	}
	return buf.String()
}
