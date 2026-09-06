package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/spf13/cobra"
)

// agent is one node of the normalized spawn-DAG relation (ADR-0042).
type agent struct {
	EvidenceScope     *agentEvidenceScope `json:"evidence_scope,omitempty"`
	ID                string              `json:"id"`
	ParentID          string              `json:"parent_id"`
	Type              string              `json:"type"`
	Started           string              `json:"started"`
	Ended             string              `json:"ended"`
	Cost              int                 `json:"cost"`
	SubtreeCost       int                 `json:"subtree_cost"`
	CostStatus        string              `json:"cost_status"`
	SubtreeCostStatus string              `json:"subtree_cost_status"`
	// Typed token components — kept separate because their economics differ
	// sharply (cache_read is ~cheap, output is ~expensive); a flat total hides
	// whether a costly-looking thread was really expensive.
	InputTokens         int    `json:"input_tokens"`
	OutputTokens        int    `json:"output_tokens"`
	CacheReadTokens     int    `json:"cache_read_tokens"`
	CacheCreationTokens int    `json:"cache_creation_tokens"`
	Status              string `json:"status"`
	Result              string `json:"result"`
}

// agentEvidenceScope names projection sources, not task outcomes or proof of
// source completeness. Only the Pi recipe populates this additive JSON object.
type agentEvidenceScope struct {
	Status              string `json:"status"`
	Timestamps          string `json:"timestamps"`
	Cost                string `json:"cost"`
	SubtreeCost         string `json:"subtree_cost"`
	TerminalRecordCount int    `json:"terminal_record_count"`
}

// addUsage accumulates a usage record's typed components into the agent and
// keeps the flat Cost total (including cache) in sync.
func (a *agent) addUsage(u usage) {
	i, o, r, c := u.typed()
	a.InputTokens += i
	a.OutputTokens += o
	a.CacheReadTokens += r
	a.CacheCreationTokens += c
	a.Cost += i + o + r + c
}

// rawRecord is the union of fields the recipes read across all schemas.
type rawRecord struct {
	Type       string          `json:"type"`
	UUID       string          `json:"uuid"`
	ParentUUID *string         `json:"parentUuid"`
	ID         string          `json:"id"`
	ParentID   string          `json:"parentId"`
	AgentID    string          `json:"agentId"`
	CustomType string          `json:"customType"`
	Timestamp  string          `json:"timestamp"`
	Message    json.RawMessage `json:"message"`
	Data       json.RawMessage `json:"data"`
}

type contentBlock struct {
	Type  string          `json:"type"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
	Text  string          `json:"text"` // pi text block: carries the "Output file: <path>" line
}

// piBackgroundDetails is the structured payload of a Pi background Agent tool-result,
// nested at message.details. The output-file path is NOT here (fullOutputPath is
// currently always null); it lives only in the tool-result's content[].text.
type piBackgroundDetails struct {
	Status         string `json:"status"`
	AgentID        string `json:"agentId"`
	SubagentType   string `json:"subagentType"`
	FullOutputPath string `json:"fullOutputPath"`
}

type message struct {
	Content json.RawMessage `json:"content"`
	Usage   usage           `json:"usage"`
	// pi background Agent tool-result: role=toolResult, toolName=Agent, structured
	// spawn state under details; the output-file path lives only in content[].text.
	Role     string              `json:"role"`
	ToolName string              `json:"toolName"`
	Details  piBackgroundDetails `json:"details"`
}

type usage struct {
	InputTokens         int `json:"input_tokens"`
	OutputTokens        int `json:"output_tokens"`
	CacheReadTokens     int `json:"cache_read_input_tokens"`     // Claude Code / sdk-cli
	CacheCreationTokens int `json:"cache_creation_input_tokens"` // Claude Code / sdk-cli
	Input               int `json:"input"`                       // pi
	Output              int `json:"output"`                      // pi
	CacheRead           int `json:"cacheRead"`                   // pi
	CacheWrite          int `json:"cacheWrite"`                  // pi
}

// typed returns the four normalized token components (fresh input, output,
// cache read, cache creation), merging the Claude Code and pi field names.
func (u usage) typed() (input, output, cacheRead, cacheCreation int) {
	input = u.InputTokens + u.Input
	output = u.OutputTokens + u.Output
	cacheRead = u.CacheReadTokens + u.CacheRead
	cacheCreation = u.CacheCreationTokens + u.CacheWrite
	return
}

// piCustomData is the self-contained pi subagent record payload.
type piCustomData struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Result      string `json:"result"`
	StartedAt   int64  `json:"startedAt"`
	CompletedAt int64  `json:"completedAt"`
}

func newTranscriptTreeCmd() *cobra.Command {
	var asJSON bool
	var strict bool
	cmd := &cobra.Command{
		Use:   "tree <session>",
		Short: "Reconstruct the spawn DAG into the normalized relation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agents, err := buildTree(args[0])
			if err != nil {
				return err
			}
			if strict {
				// strict mode (e.g. escape-hatch trust gate): abort on any violation.
				if err := validateTree(agents); err != nil {
					return err
				}
			} else {
				// navigator mode: repair orphans so a single dangling edge does not
				// make a whole session un-navigable; warn to stderr and render anyway.
				if warnings := repairTree(&agents); len(warnings) > 0 {
					for _, w := range warnings {
						fmt.Fprintln(cmd.ErrOrStderr(), "warning: "+w)
					}
					rollupSubtreeCost(agents)
				}
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(agents)
			}
			fmt.Fprint(cmd.OutOrStdout(), renderOverview(agents))
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit the normalized relation as JSON")
	cmd.Flags().BoolVar(&strict, "strict", false, "abort on validation failure instead of repairing orphans (use for untrusted/escape-hatch schemas)")
	return cmd
}

func newTranscriptShowCmd() *cobra.Command {
	var raw, asJSON bool
	var page int
	var snapshot string
	cmd := &cobra.Command{
		Use:   "show <session> <agent-id>",
		Short: "Per-agent events (meaningful by default; --raw for schema-native detail)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !asJSON && (cmd.Flags().Changed("page") || cmd.Flags().Changed("snapshot")) {
				return fmt.Errorf("transcript show: --page and --snapshot require --json")
			}
			text, err := showAgent(args[0], args[1], raw)
			if err != nil {
				return err
			}
			if !asJSON {
				fmt.Fprint(cmd.OutOrStdout(), text)
				return nil
			}
			response, err := buildTranscriptShowPage(args[0], args[1], raw, text, page, snapshot)
			if err != nil {
				return err
			}
			encoded, err := json.Marshal(response)
			if err != nil {
				return fmt.Errorf("transcript show: encode page: %w", err)
			}
			_, err = cmd.OutOrStdout().Write(append(encoded, '\n'))
			return err
		},
	}
	cmd.Flags().BoolVar(&raw, "raw", false, "emit schema-native per-agent detail (Pi: complete owned message payloads)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit one bounded lossless JSON page")
	cmd.Flags().IntVar(&page, "page", 1, "one-based page to return")
	cmd.Flags().StringVar(&snapshot, "snapshot", "", "snapshot SHA-256 returned by page 1 (required for later pages)")
	return cmd
}

// buildTree dispatches to the recipe for the session's sniffed schema and
// returns the normalized relation with subtree_cost rolled up.
func buildTree(session string) ([]agent, error) {
	schema := classifyTranscript(session)
	var agents []agent
	var err error
	switch schema {
	case schemaSDKCLI:
		agents, err = buildSDKCLITree(session)
	case schemaPi:
		agents, err = buildPiTree(session)
	case schemaClaudeCode:
		agents, err = buildClaudeCodeTree(session)
	default:
		return nil, fmt.Errorf("unknown transcript schema for %s; use the escape hatch", session)
	}
	if err != nil {
		return nil, err
	}
	initializeCostAuthority(schema, agents)
	rollupSubtreeCost(agents)
	sort.Slice(agents, func(i, j int) bool { return agents[i].ID < agents[j].ID })
	return agents, nil
}

// readRecords reads one .jsonl file into rawRecords, skipping unparseable lines.
func readRecords(path string) ([]rawRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var recs []rawRecord
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var r rawRecord
		if json.Unmarshal([]byte(line), &r) == nil {
			recs = append(recs, r)
		}
	}
	return recs, sc.Err()
}

// toolUseBlocks extracts tool_use / toolCall blocks from a message payload.
func toolUseBlocks(raw json.RawMessage) []contentBlock {
	if len(raw) == 0 {
		return nil
	}
	var msg message
	if json.Unmarshal(raw, &msg) != nil {
		return nil
	}
	var blocks []contentBlock
	_ = json.Unmarshal(msg.Content, &blocks)
	var out []contentBlock
	for _, b := range blocks {
		if b.Type == "tool_use" || b.Type == "toolCall" {
			out = append(out, b)
		}
	}
	return out
}

// piBackgroundLocator is a normalized background-spawn reference discovered from a
// Pi Agent tool-result: the child agent id and the external sidechain file that
// holds its events (path may be unreadable/expired — caller falls back to provisional).
type piBackgroundLocator struct {
	AgentID string
	Path    string // absolute, validated to stay within sessionDir; "" if none/unsafe
}

// piBackgroundLocators scans a Pi session's records for Agent tool-results carrying
// status:background + agentId + an "Output file: <path>" locator, returning one
// normalized locator per background child, carrying the RAW extracted path. Identity+
// layout authentication is deferred to read time (validatePiSidechainPath in showAgent),
// which knows the requested agent id. Discovery itself never reads a file.
func piBackgroundLocators(recs []rawRecord) []piBackgroundLocator {
	var out []piBackgroundLocator
	for _, r := range recs {
		if r.Type != "message" {
			continue
		}
		var msg message
		if json.Unmarshal(r.Message, &msg) != nil {
			continue
		}
		// structured spawn state is nested at message.details.
		if msg.Details.Status != "background" || msg.Details.AgentID == "" {
			continue
		}
		loc := piBackgroundLocator{AgentID: msg.Details.AgentID}
		// Prefer a future structured path; today it is null, so fall back to the
		// "Output file: <path>" line inside the tool-result's text content blocks.
		path := msg.Details.FullOutputPath
		if path == "" {
			var blocks []contentBlock
			_ = json.Unmarshal(msg.Content, &blocks)
			for _, b := range blocks {
				if p := parseOutputFileLocator(b.Text); p != "" {
					path = p
					break
				}
			}
		}
		// Keep the raw extracted path; identity+layout authentication happens at read
		// time in showAgent (which knows the requested agent id).
		loc.Path = strings.TrimSpace(path)
		out = append(out, loc)
	}
	return out
}

// parseOutputFileLocator extracts "<path>" from an "Output file: <path>" string.
func parseOutputFileLocator(s string) string {
	const marker = "Output file:"
	_, after, found := strings.Cut(s, marker)
	if !found {
		return ""
	}
	// Pi appends blank lines and usage instructions after the locator. The path
	// occupies only the remainder of the marker's line.
	path, _, _ := strings.Cut(after, "\n")
	return strings.TrimSpace(path)
}

// validatePiSidechainPath authenticates a Pi background sidechain path against Pi's
// task-output layout AND the requested agent identity, rather than requiring it to sit
// under the session directory (real active sidechains live in a temp
// pi-subagents-*/<session-id>/tasks/<agent-id>.output tree outside the session dir).
//
// It returns the resolved canonical path only if, after resolving symlinks and Clean:
//   - the base filename is exactly "<agentID>.output" (identity binding);
//   - the immediate parent directory is named "tasks";
//   - some ancestor directory name matches "pi-subagents-*".
//
// It rejects "" and any path that fails these — including ".." traversal (Clean/symlink
// resolution collapses it, so an arbitrary target cannot masquerade), symlink escape
// (EvalSymlinks resolves the real target before the checks), a filename for a different
// agent id, or an arbitrary file outside the pi-subagents layout. Returns "" on rejection.
func validatePiSidechainPath(p, agentID string) string {
	if p == "" || agentID == "" {
		return ""
	}
	// Resolve symlinks so the checks apply to the real target, not a link name.
	resolved, err := filepath.EvalSymlinks(p)
	if err != nil {
		return ""
	}
	clean := filepath.Clean(resolved)
	if filepath.Base(clean) != agentID+".output" {
		return ""
	}
	parent := filepath.Dir(clean)
	if filepath.Base(parent) != "tasks" {
		return ""
	}
	// Walk ancestors for a pi-subagents-* directory.
	for dir := filepath.Dir(parent); ; {
		base := filepath.Base(dir)
		if ok, _ := filepath.Match("pi-subagents-*", base); ok {
			return clean
		}
		next := filepath.Dir(dir)
		if next == dir { // reached filesystem root without a match
			return ""
		}
		dir = next
	}
}

func msgUsage(raw json.RawMessage) usage {
	var msg message
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &msg)
	}
	return msg.Usage
}

// --- sdk-cli recipe: meta.toolUseId -> tool_use block id across file union --

func buildSDKCLITree(session string) ([]agent, error) {
	base := strings.TrimSuffix(session, ".jsonl")
	subDir := filepath.Join(base, "subagents")

	// owner: tool_use block id -> agent id that issued it.
	toolOwner := map[string]string{}
	// per-agent accumulation.
	agents := map[string]*agent{}

	accumulate := func(agentID string, recs []rawRecord) {
		a := agents[agentID]
		if a == nil {
			a = &agent{ID: agentID, Type: "agent"}
			agents[agentID] = a
		}
		for _, r := range recs {
			for _, b := range toolUseBlocks(r.Message) {
				if b.ID != "" {
					toolOwner[b.ID] = agentID
				}
			}
			a.addUsage(msgUsage(r.Message))
			ts := r.Timestamp
			if ts != "" {
				if a.Started == "" || ts < a.Started {
					a.Started = ts
				}
				if ts > a.Ended {
					a.Ended = ts
				}
			}
		}
	}

	rootRecs, err := readRecords(session)
	if err != nil {
		return nil, err
	}
	accumulate("ROOT", rootRecs)

	// child agents from subagents/*.jsonl
	childMeta := map[string]string{} // agent id -> spawning toolUseId
	entries, _ := os.ReadDir(subDir)
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "agent-") && strings.HasSuffix(name, ".jsonl") {
			agentID := strings.TrimSuffix(strings.TrimPrefix(name, "agent-"), ".jsonl")
			recs, err := readRecords(filepath.Join(subDir, name))
			if err != nil {
				return nil, err
			}
			accumulate(agentID, recs)
		}
		if strings.HasPrefix(name, "agent-") && strings.HasSuffix(name, ".meta.json") {
			agentID := strings.TrimSuffix(strings.TrimPrefix(name, "agent-"), ".meta.json")
			var meta struct {
				AgentType string `json:"agentType"`
				ToolUseID string `json:"toolUseId"`
			}
			if data, err := os.ReadFile(filepath.Join(subDir, name)); err == nil {
				_ = json.Unmarshal(data, &meta)
				childMeta[agentID] = meta.ToolUseID
				if a := agents[agentID]; a != nil && meta.AgentType != "" {
					a.Type = meta.AgentType
				}
			}
		}
	}

	// resolve parent edges via meta.toolUseId -> tool_use owner.
	for agentID, toolID := range childMeta {
		if a := agents[agentID]; a != nil {
			a.ParentID = toolOwner[toolID]
		}
	}
	return mapToSlice(agents), nil
}

// --- pi recipe: Agent background tool-result -> spawn owner; terminal records -> state

func buildPiTree(session string) ([]agent, error) {
	recs, err := readRecords(session)
	if err != nil {
		return nil, err
	}
	root := &agent{ID: "ROOT", Type: "agent", EvidenceScope: &agentEvidenceScope{
		Status: "unavailable", Timestamps: "root_message_history",
		Cost: "root_message_history", SubtreeCost: "subtree_aggregate",
	}}
	agents := map[string]*agent{"ROOT": root}
	// spawn-record id (a message record hosting an Agent toolCall) -> the agent
	// that owns that record. Nested spawns live in a subagent's own records, so
	// the owner is the record's agentId (ROOT for the main stream), not always ROOT.
	recordOwner := map[string]string{}

	for _, r := range recs {
		if r.Type == "message" {
			// which agent does this record belong to? agentId names a subagent;
			// its absence means the main (ROOT) stream.
			owner := "ROOT"
			if r.AgentID != "" {
				owner = r.AgentID
			}
			if owner == "ROOT" {
				root.addUsage(msgUsage(r.Message))
				if r.Timestamp != "" {
					if root.Started == "" || r.Timestamp < root.Started {
						root.Started = r.Timestamp
					}
					if r.Timestamp > root.Ended {
						root.Ended = r.Timestamp
					}
				}
			}
			// a record hosting an Agent toolCall is a spawn point owned by `owner`.
			for _, b := range toolUseBlocks(r.Message) {
				if b.Name == "Agent" {
					recordOwner[r.ID] = owner
				}
			}
		}
	}

	// Background Agent tool-results carry the child identity. Resolve their
	// owning spawn record before terminal records are applied: terminal parentId
	// is event sequencing and may point at a preceding completion event.
	spawnParentByAgentID := map[string]string{}
	for _, r := range recs {
		if r.Type != "message" {
			continue
		}
		var msg message
		if json.Unmarshal(r.Message, &msg) != nil || msg.Details.Status != "background" || msg.Details.AgentID == "" {
			continue
		}
		parent := "ROOT"
		if owner, ok := recordOwner[r.ParentID]; ok {
			parent = owner
		}
		spawnParentByAgentID[msg.Details.AgentID] = parent
	}

	// Terminal records supersede provisional status/result, but never parentage.
	terminalCounts := map[string]int{}
	for _, r := range recs {
		if r.Type == "custom" && r.CustomType == "subagents:record" {
			var d piCustomData
			_ = json.Unmarshal(r.Data, &d)
			parent, ok := spawnParentByAgentID[d.ID]
			if !ok {
				// Older inline records may point directly at the spawning Agent call.
				// With no authenticated spawn evidence, attach conservatively to ROOT
				// rather than exposing a transcript event id as an agent parent.
				parent = "ROOT"
				if owner, found := recordOwner[r.ParentID]; found {
					parent = owner
				}
			}
			terminalCounts[d.ID]++
			a := &agent{
				EvidenceScope: &agentEvidenceScope{
					Status: "last_terminal_record", Timestamps: "last_terminal_record",
					Cost: "unavailable", SubtreeCost: "subtree_aggregate",
					TerminalRecordCount: terminalCounts[d.ID],
				},
				ID:       d.ID,
				Type:     d.Type,
				Status:   d.Status,
				Result:   d.Result,
				ParentID: parent,
			}
			if d.Type == "" {
				a.Type = "agent"
			}
			if d.StartedAt != 0 {
				a.Started = time.UnixMilli(d.StartedAt).UTC().Format(time.RFC3339)
			}
			if d.CompletedAt != 0 {
				a.Ended = time.UnixMilli(d.CompletedAt).UTC().Format(time.RFC3339)
			}
			agents[d.ID] = a
		}
	}

	// background children: discovered from Agent tool-result records (structured spawn
	// state at message.details). A terminal subagents:record (added above) supersedes,
	// so only add ids not already present.
	for _, r := range recs {
		if r.Type != "message" {
			continue
		}
		var msg message
		if json.Unmarshal(r.Message, &msg) != nil {
			continue
		}
		if msg.Details.Status != "background" || msg.Details.AgentID == "" {
			continue
		}
		if _, ok := agents[msg.Details.AgentID]; ok {
			continue // terminal record supersedes provisional spawn state
		}
		parent := "ROOT"
		if owner, ok := recordOwner[r.ParentID]; ok {
			parent = owner
		}
		typ := msg.Details.SubagentType
		if typ == "" {
			typ = "agent"
		}
		agents[msg.Details.AgentID] = &agent{
			EvidenceScope: &agentEvidenceScope{
				Status: "background_spawn_record", Timestamps: "unavailable",
				Cost: "unavailable", SubtreeCost: "subtree_aggregate",
			},
			ID:       msg.Details.AgentID,
			Type:     typ,
			Status:   "background",
			ParentID: parent,
		}
	}

	// Attribute usage from each readable authenticated sidechain exactly once.
	// Both the path identity and each record's agentId must match, so nested agent
	// events cannot leak into their parent's own cost.
	hydrated := map[string]bool{}
	for _, loc := range piBackgroundLocators(recs) {
		if hydrated[loc.AgentID] {
			continue
		}
		a := agents[loc.AgentID]
		if a == nil {
			continue
		}
		safe := validatePiSidechainPath(loc.Path, loc.AgentID)
		if safe == "" {
			continue
		}
		side, err := readRecords(safe)
		if err != nil {
			continue
		}
		matched := false
		for _, sr := range side {
			// Native tool results are retrievable events, not model usage evidence.
			if !isPiEventRecord(sr) || sr.Type == "toolResult" || sr.AgentID != loc.AgentID {
				continue
			}
			matched = true
			a.addUsage(msgUsage(sr.Message))
		}
		if matched {
			a.CostStatus = "complete"
			a.EvidenceScope.Cost = "retained_sidechain_history"
			hydrated[loc.AgentID] = true
		}
	}

	return mapToSlice(agents), nil
}

// --- claude-code recipe: inline Task; parent = assistant turn issuing Task ---

func buildClaudeCodeTree(session string) ([]agent, error) {
	recs, err := readRecords(session)
	if err != nil {
		return nil, err
	}
	root := &agent{ID: "ROOT", Type: "agent"}
	agents := map[string]*agent{"ROOT": root}
	for _, r := range recs {
		if r.Type == "assistant" {
			root.addUsage(msgUsage(r.Message))
			if r.Timestamp != "" {
				if root.Started == "" || r.Timestamp < root.Started {
					root.Started = r.Timestamp
				}
				if r.Timestamp > root.Ended {
					root.Ended = r.Timestamp
				}
			}
			// each inline Task tool_use spawns a child; the child id is the
			// tool_use id and its parent is ROOT (the turn issuing it).
			for _, b := range toolUseBlocks(r.Message) {
				if b.Name == "Task" {
					agents[b.ID] = &agent{ID: b.ID, Type: "agent", ParentID: "ROOT", Started: r.Timestamp}
				}
			}
		}
	}
	return mapToSlice(agents), nil
}

func mapToSlice(m map[string]*agent) []agent {
	out := make([]agent, 0, len(m))
	for _, a := range m {
		out = append(out, *a)
	}
	return out
}

func initializeCostAuthority(schema string, agents []agent) {
	for i := range agents {
		switch schema {
		case schemaSDKCLI:
			agents[i].CostStatus = "complete"
		case schemaClaudeCode:
			if agents[i].ID == "ROOT" {
				agents[i].CostStatus = "complete"
			} else {
				agents[i].CostStatus = "unavailable"
			}
		case schemaPi:
			if agents[i].ID == "ROOT" {
				agents[i].CostStatus = "complete"
			} else if agents[i].CostStatus == "" {
				agents[i].CostStatus = "unavailable"
			}
		}
	}
}

// rollupSubtreeCost sets each agent's subtree_cost = own cost + descendants
// and propagates whether every included own-cost measurement is complete.
func rollupSubtreeCost(agents []agent) {
	children := map[string][]int{} // parent id -> indices
	idIndex := map[string]int{}
	for i := range agents {
		idIndex[agents[i].ID] = i
	}
	for i := range agents {
		if agents[i].ParentID != "" {
			children[agents[i].ParentID] = append(children[agents[i].ParentID], i)
		}
	}
	var sum func(i int) (int, bool)
	seen := map[int]bool{}
	sum = func(i int) (int, bool) {
		if seen[i] {
			return agents[i].Cost, agents[i].CostStatus == "complete" // cycle guard; validation reports the real error
		}
		seen[i] = true
		total := agents[i].Cost
		complete := agents[i].CostStatus == "complete"
		for _, ci := range children[agents[i].ID] {
			childTotal, childComplete := sum(ci)
			total += childTotal
			complete = complete && childComplete
		}
		agents[i].SubtreeCost = total
		if complete {
			agents[i].SubtreeCostStatus = "complete"
		} else {
			agents[i].SubtreeCostStatus = "partial"
		}
		return total, complete
	}
	for i := range agents {
		if agents[i].ParentID == "" {
			sum(i)
		}
	}
	// agents whose parent was never a root-descendant still need a subtree value
	for i := range agents {
		if !seen[i] {
			sum(i)
		}
	}
}

// repairTree makes a malformed spawn tree navigable: any agent whose parent_id
// does not resolve to a present agent is re-parented to ROOT (creating a
// synthetic ROOT if none exists), and cycles are broken by detaching the
// back-edge. It returns a human-readable warning per repair so the problem is
// surfaced rather than hidden. Used in navigator mode; --strict skips it.
func repairTree(agentsPtr *[]agent) []string {
	agents := *agentsPtr
	ids := map[string]bool{}
	for i := range agents {
		ids[agents[i].ID] = true
	}
	var warnings []string
	// ensure a ROOT to re-home orphans under.
	rootID := "ROOT"
	if !ids["ROOT"] {
		agents = append(agents, agent{ID: "ROOT", Type: "agent"})
		ids["ROOT"] = true
		*agentsPtr = agents
	}
	for i := range agents {
		p := agents[i].ParentID
		if p == "" || ids[p] {
			continue
		}
		warnings = append(warnings, fmt.Sprintf("agent %q had unresolved parent %q; re-parented to %s", agents[i].ID, p, rootID))
		agents[i].ParentID = rootID
	}
	// break any remaining cycle by detaching the offending edge to ROOT.
	if cyc := findCycleNode(agents); cyc != "" {
		for i := range agents {
			if agents[i].ID == cyc {
				warnings = append(warnings, fmt.Sprintf("agent %q was in a cycle; detached to %s", cyc, rootID))
				agents[i].ParentID = ""
			}
		}
	}
	return warnings
}

// anyRecordForAgent reports whether any renderable Pi event is owned by agentID.
func anyRecordForAgent(recs []rawRecord, agentID string) bool {
	for _, r := range recs {
		if isPiEventRecord(r) && r.AgentID == agentID && agentID != "" {
			return true
		}
	}
	return false
}

func isPiEventRecord(r rawRecord) bool {
	return isPiEventType(r.Type)
}

func isPiEventType(kind string) bool {
	return kind == "message" || kind == "assistant" || kind == "user" || kind == "toolResult"
}

// findCycleNode returns the id of a node participating in a cycle, or "".
func findCycleNode(agents []agent) string {
	childIdx := map[string][]string{}
	for _, a := range agents {
		if a.ParentID != "" {
			childIdx[a.ParentID] = append(childIdx[a.ParentID], a.ID)
		}
	}
	color := map[string]int{}
	var found string
	var visit func(id string) bool
	visit = func(id string) bool {
		color[id] = 1
		for _, c := range childIdx[id] {
			if color[c] == 1 {
				found = c
				return true
			}
			if color[c] == 0 && visit(c) {
				return true
			}
		}
		color[id] = 2
		return false
	}
	for _, a := range agents {
		if color[a.ID] == 0 {
			if visit(a.ID) {
				return found
			}
		}
	}
	return ""
}

// validateTree enforces the ADR-0042 assertions before output.
func validateTree(agents []agent) error {
	ids := map[string]bool{}
	for _, a := range agents {
		ids[a.ID] = true
	}
	roots := 0
	for _, a := range agents {
		if a.ParentID == "" {
			roots++
			continue
		}
		if !ids[a.ParentID] {
			return fmt.Errorf("validation failed: agent %q has unresolved parent %q", a.ID, a.ParentID)
		}
	}
	if roots == 0 && len(agents) > 0 {
		return fmt.Errorf("validation failed: no root agent (every agent has a parent — cycle)")
	}
	// cycle detection via DFS
	color := map[string]int{} // 0 unvisited, 1 in-stack, 2 done
	childIdx := map[string][]string{}
	for _, a := range agents {
		if a.ParentID != "" {
			childIdx[a.ParentID] = append(childIdx[a.ParentID], a.ID)
		}
	}
	var visit func(id string) error
	visit = func(id string) error {
		color[id] = 1
		for _, c := range childIdx[id] {
			switch color[c] {
			case 1:
				return fmt.Errorf("validation failed: cycle detected at agent %q", c)
			case 0:
				if err := visit(c); err != nil {
					return err
				}
			}
		}
		color[id] = 2
		return nil
	}
	for _, a := range agents {
		if color[a.ID] == 0 {
			if err := visit(a.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

// renderOverview draws the descent-stack lifespan tree (text-first).
func renderOverview(agents []agent) string {
	children := map[string][]agent{}
	byID := map[string]agent{}
	for _, a := range agents {
		byID[a.ID] = a
		children[a.ParentID] = append(children[a.ParentID], a)
	}
	for k := range children {
		sort.Slice(children[k], func(i, j int) bool {
			return children[k][i].Started < children[k][j].Started
		})
	}
	var b strings.Builder
	var walk func(id string, depth int)
	walk = func(id string, depth int) {
		a := byID[id]
		indent := strings.Repeat("  ", depth)
		status := a.Status
		if status == "" {
			status = "—"
		}
		cost := fmt.Sprintf("%d", a.Cost)
		if a.CostStatus != "complete" {
			cost = "?"
		}
		subtree := fmt.Sprintf("%d", a.SubtreeCost)
		if a.SubtreeCostStatus != "complete" {
			subtree = "≥" + subtree
		}
		fmt.Fprintf(&b, "%s%s  [%s]  cost=%s subtree=%s  %s\n",
			indent, a.ID, a.Type, cost, subtree, status)
		for _, c := range children[id] {
			walk(c.ID, depth+1)
		}
	}
	// roots (parent_id == "")
	for _, a := range agents {
		if a.ParentID == "" {
			walk(a.ID, 0)
		}
	}
	return b.String()
}

// renderPiEvents applies the selected mode consistently to already-owned events.
func renderPiEvents(b *strings.Builder, recs []rawRecord, raw bool) {
	if !raw {
		renderMeaningfulEvents(b, recs)
		return
	}
	for _, r := range recs {
		fmt.Fprintf(b, "%s\n", string(r.Message))
	}
}

// showAgent surfaces per-agent events for one agent id. By default it renders
// only meaningful events; raw=true emits schema-native per-agent detail.
func showAgent(session, agentID string, raw bool) (string, error) {
	schema := classifyTranscript(session)
	var b strings.Builder
	fmt.Fprintf(&b, "agent %s (schema %s)\n", agentID, schema)

	// For pi, the agent may be a custom subagents:record — surface its full data.
	if schema == schemaPi {
		recs, err := readRecords(session)
		if err != nil {
			return "", err
		}
		// ROOT (or a subagent) main-stream events: render the message records
		// owned by that agent (agentId names a subagent; empty = ROOT stream).
		if agentID == "ROOT" || anyRecordForAgent(recs, agentID) {
			var owned []rawRecord
			for _, r := range recs {
				if !isPiEventRecord(r) {
					continue
				}
				owner := "ROOT"
				if r.AgentID != "" {
					owner = r.AgentID
				}
				if owner == agentID {
					owned = append(owned, r)
				}
			}
			if len(owned) > 0 {
				renderPiEvents(&b, owned, raw)
				return b.String(), nil
			}
		}
		// Retain terminal metadata as fallback, but prefer a readable authenticated
		// sidechain because it contains the agent's full meaningful event history.
		var terminal *piCustomData
		var terminalRaw json.RawMessage
		for _, r := range recs {
			if r.Type == "custom" && r.CustomType == "subagents:record" {
				var d piCustomData
				_ = json.Unmarshal(r.Data, &d)
				if d.ID == agentID {
					terminal = &d
					terminalRaw = r.Data
					break
				}
			}
		}
		for _, loc := range piBackgroundLocators(recs) {
			if loc.AgentID != agentID {
				continue
			}
			// Authenticate the path against Pi's tasks/<agentID>.output layout before
			// reading — accepts the real out-of-session pi-subagents-* location and
			// rejects traversal/symlink-escape/mismatched-id/arbitrary files.
			if safe := validatePiSidechainPath(loc.Path, agentID); safe != "" {
				if side, err := readRecords(safe); err == nil {
					// An authenticated filename does not establish event ownership.
					// Match the same explicit owner used for sidechain usage attribution;
					// never apply the main-stream empty-owner-to-ROOT convention here.
					var owned []rawRecord
					for _, r := range side {
						if isPiEventRecord(r) && r.AgentID == agentID {
							owned = append(owned, r)
						}
					}
					if len(owned) > 0 {
						if terminal != nil {
							fmt.Fprintf(&b, "type: %s\nstatus: %s\n", terminal.Type, terminal.Status)
						}
						renderPiEvents(&b, owned, raw)
						return b.String(), nil
					}
				}
			}
			if terminal == nil {
				// File missing/expired/unauthenticated: provisional metadata, not an error.
				fmt.Fprintf(&b, "status: background\n(sidechain output unavailable for %s)\n", agentID)
				return b.String(), nil
			}
			break
		}
		if terminal != nil {
			fmt.Fprintf(&b, "type: %s\nstatus: %s\nresult: %s\n", terminal.Type, terminal.Status, terminal.Result)
			fmt.Fprintf(&b, "raw: %s\n", string(terminalRaw))
			return b.String(), nil
		}
	}

	// sdk-cli: render the agent's meaningful events only. Attachments
	// (deferred_tools_delta, skill_listing) and tool-result payloads are noise
	// for the debug/recover use case and are omitted; use --raw for the full file.
	if schema == schemaSDKCLI {
		base := strings.TrimSuffix(session, ".jsonl")
		file := filepath.Join(base, "subagents", "agent-"+agentID+".jsonl")
		if raw {
			if data, err := os.ReadFile(file); err == nil {
				b.Write(data)
				return b.String(), nil
			}
			fmt.Fprintf(&b, "(no per-agent detail found for %s)\n", agentID)
			return b.String(), nil
		}
		recs, err := readRecords(file)
		if err != nil {
			fmt.Fprintf(&b, "(no per-agent detail found for %s)\n", agentID)
			return b.String(), nil
		}
		renderMeaningfulEvents(&b, recs)
		return b.String(), nil
	}

	fmt.Fprintf(&b, "(no per-agent detail found for %s)\n", agentID)
	return b.String(), nil
}

// textContent extracts a plain-text summary from a message content payload,
// which may be a bare string or an array of typed blocks.
func textContent(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var blocks []struct {
		Type  string          `json:"type"`
		Text  string          `json:"text"`
		Name  string          `json:"name"`
		Input json.RawMessage `json:"input"`
	}
	if json.Unmarshal(raw, &blocks) != nil {
		return ""
	}
	var parts []string
	for _, bl := range blocks {
		switch bl.Type {
		case "text":
			if bl.Text != "" {
				parts = append(parts, bl.Text)
			}
		case "tool_use", "toolCall":
			// tool CALL: keep the name and a brief arg preview; drop nothing else.
			arg := strings.TrimSpace(string(bl.Input))
			if len(arg) > 120 {
				end := 120
				for end > 0 && !utf8.RuneStart(arg[end]) {
					end--
				}
				arg = arg[:end] + "…"
			}
			parts = append(parts, fmt.Sprintf("→ %s(%s)", bl.Name, arg))
		case "tool_result":
			// tool RESULT payloads are omitted for this use case.
		}
	}
	return strings.Join(parts, "\n")
}

// meaningfulContent is the shared search/show content policy. Tool-result
// payloads require raw mode even when encoded as ordinary text blocks.
func meaningfulContent(role string, content json.RawMessage) string {
	if role == "toolResult" || role == "tool_result" {
		return ""
	}
	return textContent(content)
}

// renderMeaningfulEvents writes prompt / assistant text / tool calls in order,
// skipping attachment records and tool-result payloads.
func renderMeaningfulEvents(b *strings.Builder, recs []rawRecord) {
	for _, r := range recs {
		switch r.Type {
		case "attachment":
			// noise: deferred_tools_delta, skill_listing, file snapshots.
			continue
		case "user", "assistant", "message":
			// "user"/"assistant" = Claude Code / sdk-cli; "message" = pi.
			var msg struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			}
			if json.Unmarshal(r.Message, &msg) != nil {
				continue
			}
			text := meaningfulContent(msg.Role, msg.Content)
			if strings.TrimSpace(text) == "" {
				continue
			}
			role := msg.Role
			if role == "" {
				role = r.Type
			}
			fmt.Fprintf(b, "[%s] %s\n", role, text)
		}
	}
}
