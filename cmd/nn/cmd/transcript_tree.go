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

	"github.com/spf13/cobra"
)

// agent is one node of the normalized spawn-DAG relation (ADR-0042).
type agent struct {
	ID          string `json:"id"`
	ParentID    string `json:"parent_id"`
	Type        string `json:"type"`
	Started     string `json:"started"`
	Ended       string `json:"ended"`
	Cost        int    `json:"cost"`
	SubtreeCost int    `json:"subtree_cost"`
	Status      string `json:"status"`
	Result      string `json:"result"`
}

// rawRecord is the union of fields the recipes read across all schemas.
type rawRecord struct {
	Type       string          `json:"type"`
	UUID       string          `json:"uuid"`
	ParentUUID *string         `json:"parentUuid"`
	ID         string          `json:"id"`
	ParentID   string          `json:"parentId"`
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
}

type message struct {
	Content json.RawMessage `json:"content"`
	Usage   usage           `json:"usage"`
}

type usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	Input        int `json:"input"`  // pi
	Output       int `json:"output"` // pi
}

func (u usage) tokens() int {
	return u.InputTokens + u.OutputTokens + u.Input + u.Output
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
	cmd := &cobra.Command{
		Use:   "tree <session>",
		Short: "Reconstruct the spawn DAG into the normalized relation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agents, err := buildTree(args[0])
			if err != nil {
				return err
			}
			if err := validateTree(agents); err != nil {
				return err
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
	return cmd
}

func newTranscriptShowCmd() *cobra.Command {
	var raw bool
	cmd := &cobra.Command{
		Use:   "show <session> <agent-id>",
		Short: "Per-agent events (meaningful by default; --raw for the lossless full record)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			text, err := showAgent(args[0], args[1], raw)
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), text)
			return nil
		},
	}
	cmd.Flags().BoolVar(&raw, "raw", false, "emit the lossless full per-agent record (all events, verbatim)")
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
			a.Cost += msgUsage(r.Message).tokens()
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

// --- pi recipe: custom(subagents:record).parentId -> Agent toolCall record id

func buildPiTree(session string) ([]agent, error) {
	recs, err := readRecords(session)
	if err != nil {
		return nil, err
	}
	// map: record id -> the agent that record belongs to (ROOT for main thread).
	// pi keeps the whole session (incl. Agent toolCalls) in the root stream.
	root := &agent{ID: "ROOT", Type: "agent"}
	agents := map[string]*agent{"ROOT": root}
	// toolCall record id (the message record carrying an Agent toolCall) -> ROOT
	recordOwner := map[string]string{}

	for _, r := range recs {
		if r.Type == "message" {
			root.Cost += msgUsage(r.Message).tokens()
			if r.Timestamp != "" {
				if root.Started == "" || r.Timestamp < root.Started {
					root.Started = r.Timestamp
				}
				if r.Timestamp > root.Ended {
					root.Ended = r.Timestamp
				}
			}
			// a message record hosting an Agent toolCall is a spawn point;
			// its record id owns the spawn.
			for _, b := range toolUseBlocks(r.Message) {
				if b.Name == "Agent" {
					recordOwner[r.ID] = "ROOT"
				}
			}
		}
	}

	// subagent records
	for _, r := range recs {
		if r.Type == "custom" && r.CustomType == "subagents:record" {
			var d piCustomData
			_ = json.Unmarshal(r.Data, &d)
			// parentId must resolve to a spawning Agent toolCall record owner.
			// If it does not, preserve the dangling id so validation reports the
			// unresolved edge rather than silently promoting the agent to a root.
			parent := r.ParentID
			if owner, ok := recordOwner[r.ParentID]; ok {
				parent = owner
			}
			a := &agent{
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
			root.Cost += msgUsage(r.Message).tokens()
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

// rollupSubtreeCost sets each agent's subtree_cost = own cost + descendants.
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
	var sum func(i int) int
	seen := map[int]bool{}
	sum = func(i int) int {
		if seen[i] {
			return agents[i].Cost // cycle guard; validation reports the real error
		}
		seen[i] = true
		total := agents[i].Cost
		for _, ci := range children[agents[i].ID] {
			total += sum(ci)
		}
		agents[i].SubtreeCost = total
		return total
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
		fmt.Fprintf(&b, "%s%s  [%s]  cost=%d subtree=%d  %s\n",
			indent, a.ID, a.Type, a.Cost, a.SubtreeCost, status)
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

// showAgent surfaces per-agent events for one agent id. By default it renders
// only meaningful events; raw=true emits the lossless full record.
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
		for _, r := range recs {
			if r.Type == "custom" && r.CustomType == "subagents:record" {
				var d piCustomData
				_ = json.Unmarshal(r.Data, &d)
				if d.ID == agentID {
					fmt.Fprintf(&b, "type: %s\nstatus: %s\nresult: %s\n", d.Type, d.Status, d.Result)
					fmt.Fprintf(&b, "raw: %s\n", string(r.Data))
					return b.String(), nil
				}
			}
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
		Type string          `json:"type"`
		Text string          `json:"text"`
		Name string          `json:"name"`
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
				arg = arg[:120] + "…"
			}
			parts = append(parts, fmt.Sprintf("→ %s(%s)", bl.Name, arg))
		case "tool_result":
			// tool RESULT payloads are omitted for this use case.
		}
	}
	return strings.Join(parts, "\n")
}

// renderMeaningfulEvents writes prompt / assistant text / tool calls in order,
// skipping attachment records and tool-result payloads.
func renderMeaningfulEvents(b *strings.Builder, recs []rawRecord) {
	for _, r := range recs {
		switch r.Type {
		case "attachment":
			// noise: deferred_tools_delta, skill_listing, file snapshots.
			continue
		case "user", "assistant":
			var msg struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			}
			if json.Unmarshal(r.Message, &msg) != nil {
				continue
			}
			text := textContent(msg.Content)
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
