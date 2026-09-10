package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/jaresty/nn/internal/attention"
)

type observeLiveAgent struct {
	TailSnapshot string              `json:"tail_snapshot,omitempty"`
	MetadataKey  string              `json:"metadata_key,omitempty"`
	SourceStamp  *observeSourceStamp `json:"source_stamp,omitempty"`
	ID           string              `json:"id"`
	Fingerprint  string              `json:"fingerprint"`
	State        string              `json:"state"`
	Snapshot     string              `json:"attention_snapshot,omitempty"`
	Room         *attentionRoom      `json:"result,omitempty"`
}
type observeLiveState struct {
	WorkerScans    map[string]int       `json:"worker_scans,omitempty"`
	ReadableOffset int                  `json:"readable_offset,omitempty"`
	DecodedRecords *uint64              `json:"decoded_records,omitempty"`
	Sources        []observeSourceStamp `json:"sources,omitempty"`
	Authorities    []observeAuthority   `json:"authorities,omitempty"`
	OptionsKey     string               `json:"options_key,omitempty"`
	CanonicalPath  string               `json:"canonical_path,omitempty"`
	Version        int                  `json:"version"`
	Task           string               `json:"task"`
	AgentTasks     []string             `json:"agent_tasks,omitempty"`
	Path           string               `json:"path"`
	Recent         time.Duration        `json:"recent"`
	At             time.Time            `json:"at"`
	Agents         []observeLiveAgent   `json:"agents"`
	Text           string               `json:"text"`
}

func observationText(s observeLiveState, id string) string {
	diagnostic := ""
	if s.DecodedRecords != nil {
		diagnostic = fmt.Sprintf("Acquisition: %d raw transcript records decoded, including repeated reads; replay retains this measurement.\n", *s.DecodedRecords)
	}
	if len(s.WorkerScans) > 0 {
		diagnostic += fmt.Sprintf("Worker acquisition: %v; verified append still hashes prior bytes; replay retains this measurement.\n", s.WorkerScans)
	}
	return fmt.Sprintf("Observation snapshot: %s\n"+diagnostic+"Refresh: nn transcript observe %q --refresh %s\nCoverage: nn transcript observe %q --snapshot %s --coverage-page 1\n\n", id, s.Path, id, s.Path, id) + s.Text
}
func loadObserveState(snapshot, session string) (observeLiveState, error) {
	var s observeLiveState
	if !validCaptureID(snapshot) {
		return s, fmt.Errorf("observe: invalid snapshot")
	}
	b, err := readCaptureFile(snapshot + ".observe")
	if err != nil {
		return s, fmt.Errorf("observe: prior state unavailable; omit --refresh/--snapshot for an explicit fresh baseline: %w", err)
	}
	if captureHash(b) != snapshot || json.Unmarshal(b, &s) != nil || s.Version != 1 {
		return s, fmt.Errorf("observe: corrupt state; explicitly start a fresh baseline")
	}
	path, err := filepath.Abs(session)
	if err != nil {
		return s, err
	}
	if path != s.Path {
		return s, fmt.Errorf("observe: snapshot scope mismatch")
	}
	return s, nil
}
func saveObserveState(s observeLiveState) (string, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return "", err
	}
	if len(b) > 4*1024*1024 {
		return "", fmt.Errorf("observe: retained state exceeds 4 MiB; no complete observation published")
	}
	id := captureHash(b)
	return id, writeCaptureFile(id+".observe", b)
}

// Hash complete owned evidence and complete launch invocations, not clipped task previews.
func observeAgentEvidence(a agent, records []ledgerRecord, handoffs []piHandoff, detail, task string, policies []*attention.Policy, ownedEvidencePresent ...bool) (string, time.Time, bool, error) {
	parts := []any{a.ID, a.ParentID, a.Status, a.Description, a.Started, a.Ended, a.Result, detail, task, policies}
	latest := time.Time{}
	unknown := len(records) == 0 && !(len(ownedEvidencePresent) > 0 && ownedEvidencePresent[0])
	add := func(r rawRecord, path string) {
		parts = append(parts, []any{path, r.RecordOrdinal, r})
		var stamp any = r.Timestamp
		if r.Timestamp == "" {
			var header struct {
				Timestamp any `json:"timestamp"`
			}
			_ = json.Unmarshal(r.Message, &header)
			stamp = header.Timestamp
		}
		tm, status := ledgerTime(stamp)
		if status != "known" {
			unknown = true
		} else if tm.After(latest) {
			latest = tm
		}
	}
	for _, r := range records {
		add(r.Record, r.Path)
	}
	for _, h := range handoffs {
		if h.Child == a.ID {
			add(h.Record, "parent")
			if h.Invocation != nil {
				parts = append(parts, h.Invocation.Raw)
				add(h.Invocation.Record, "parent")
			}
		}
	}
	h := sha256.New()
	encoder := json.NewEncoder(h)
	for _, part := range parts {
		if err := encoder.Encode(part); err != nil {
			return "", latest, unknown, err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), latest, unknown, nil
}
func observeEligible(prior *observeLiveAgent, fingerprint string, latest time.Time, unknown bool, cutoff time.Time, refresh bool) bool {
	if refresh {
		return prior == nil || prior.Fingerprint != fingerprint
	}
	return unknown || !latest.Before(cutoff)
}

func observeLiveSection(session string, o observeAttentionOptions, recent time.Duration, previous string, sharedParent ...*transcriptCapture) (string, *observeLiveState, error) {
	path, err := filepath.Abs(session)
	if err != nil {
		return "", nil, err
	}
	prior := map[string]observeLiveAgent{}
	if previous != "" {
		s, e := loadObserveState(previous, session)
		if e != nil {
			return "", nil, e
		}
		if recent != s.Recent {
			return "", nil, fmt.Errorf("observe: Refresh recency differs from prior state; start a fresh baseline to change it")
		}
		for _, a := range s.Agents {
			prior[a.ID] = a
		}
	}
	state := &observeLiveState{Version: 1, Path: path, At: time.Now().UTC(), Recent: recent, Task: o.Task, AgentTasks: o.AgentTasks}
	state.OptionsKey, err = observeOptionsKey(o, recent)
	if err != nil {
		return "", nil, err
	}
	policies, err := attention.Builtins()
	if err != nil {
		return "", nil, err
	}
	overrides, err := parseAgentTasks(o.AgentTasks)
	if err != nil {
		return "", nil, err
	}
	var capture *transcriptCapture
	var agents []agent
	var handoffs []piHandoff
	sidePaths := map[string]string{}
	canonical, err := contextPath(session)
	if err != nil {
		return "", nil, err
	}
	state.CanonicalPath = canonical
	schema := classifyTranscript(canonical)
	if schema == schemaPi {
		if len(sharedParent) > 0 && sharedParent[0] != nil {
			capture = sharedParent[0]
			if capture.Path != canonical {
				return "", nil, fmt.Errorf("observe: shared parent scope mismatch")
			}
		} else {
			capture, err = observeParentCapture(session, canonical)
		}
		if err != nil {
			return "", nil, err
		}
		agents, err = buildPiTreeUsing(capture.Path, capture.read, capture.resolve)
		if err == nil {
			parent, e := capture.read(canonical)
			if e != nil {
				return "", nil, e
			}
			handoffs = piHandoffs(parent)
			for _, loc := range piBackgroundLocators(parent) {
				if _, exists := sidePaths[loc.AgentID]; !exists {
					sidePaths[loc.AgentID] = loc.Path
				}
			}
		}
	} else {
		agents, err = buildTree(session)
	}
	if err != nil {
		return "", nil, err
	}
	sort.Slice(agents, func(i, j int) bool {
		if agents[i].ID == "ROOT" {
			return true
		}
		if agents[j].ID == "ROOT" {
			return false
		}
		return agents[i].ID < agents[j].ID
	})
	known := map[string]bool{}
	for _, a := range agents {
		known[a.ID] = true
	}
	for id := range overrides {
		if !known[id] {
			return "", nil, fmt.Errorf("observe: agent-task names unknown agent %q", id)
		}
	}
	batch := attentionRetention{InputPath: path, Page: attentionPage{Version: 2, MetricVersion: 2, Path: canonical, Schema: schema, Task: o.Task, AgentTasks: overrides, Policy: policies[0], Policies: policies, Population: len(agents), Limitations: attentionLimitations, CaptureMode: "sequential owned-record projections"}, Evidence: map[string][]attentionEvidence{}}
	if capture != nil {
		batch.Page.CaptureID = capture.ID
		batch.Page.CaptureMode = "sequential owned-record projections; shared parent assignment evidence"
	}
	details := []attentionPage{}
	displayed := 0
	flush := func() error {
		if len(batch.Page.Rooms) == 0 {
			return nil
		}
		batch.Page.Evaluated = len(batch.Page.Rooms)
		batch.Page.SelectedAgents = batch.Page.Evaluated
		batch.Page.Unevaluated = len(agents) - batch.Page.Evaluated
		r, e := retainAttention(batch)
		if e != nil {
			return e
		}
		for i := range state.Agents {
			if state.Agents[i].Room != nil && state.Agents[i].Snapshot == "" {
				state.Agents[i].Snapshot = r.Page.Snapshot
			}
		}
		if displayed < o.Limit {
			p := r.Page
			n := o.Limit - displayed
			if n < len(p.Rooms) {
				p.Rooms = p.Rooms[:n]
			}
			p.Evaluated = len(p.Rooms)
			p.SelectedAgents = p.Evaluated
			p.Unevaluated = len(agents) - p.Evaluated
			details = append(details, p)
			displayed += len(p.Rooms)
		}
		batch.Page.Rooms = nil
		batch.Page.SignalEvaluations = 0
		batch.Evidence = map[string][]attentionEvidence{}
		return nil
	}
	counts := map[string]int{}
	outcomes := map[string]int{}
	evaluated := 0
	workLimit := 0
	for _, policy := range policies {
		if policy.Window.LastWorkEvents > workLimit {
			workLimit = policy.Window.LastWorkEvents
		}
	}
	state.WorkerScans = map[string]int{}
	unknownInspections := 0
agentsLoop:
	for _, a := range agents {
		old, exists := prior[a.ID]
		task, source := o.Task, "cohort_override"
		if v, ok := overrides[a.ID]; ok {
			task, source = v, "agent_override"
		}
		metadataKey := ""
		var sourceStamp *observeSourceStamp
		var tail *observeTail
		var parentRecords []ledgerRecord
		tailSnapshot := ""
		var records []ledgerRecord
		var detail string
		var acquisitionErr error
		metadataRecent := false
		if capture != nil {
			records, detail = capture.ledger(a.ID)
			parentRecords = append([]ledgerRecord{}, records...)
			var parentLatest time.Time
			var parentUnknown bool
			parentEvidence := len(records) > 0
			for _, h := range handoffs {
				if h.Child == a.ID {
					parentEvidence = true
					break
				}
			}
			metadataKey, parentLatest, parentUnknown, err = observeAgentEvidence(a, records, handoffs, "parent-metadata:"+source, task, policies, parentEvidence)
			if err != nil {
				return "", nil, err
			}
			inline := false
			for _, r := range records {
				if !r.Lifecycle {
					inline = true
					break
				}
			}
			if !inline {
				if locPath, exists := sidePaths[a.ID]; exists {
					safe := validatePiSidechainPath(locPath, a.ID)
					if safe != "" {
						capture.AuthPaths[locPath+"\x00"+a.ID] = safe
						stamp, stampErr := currentObserveStamp(safe)
						if stampErr == nil && exists && old.MetadataKey == metadataKey && old.SourceStamp != nil && sameObserveStamp(*old.SourceStamp, stamp) {
							state.Sources = append(state.Sources, stamp)
							if old.Room != nil {
								old.State = "unchanged"
							}
							counts[old.State]++
							state.Agents = append(state.Agents, old)
							continue agentsLoop
						}
						if previous == "" {
							clock := observeWorkerRecency(stamp, stampErr, parentLatest, parentUnknown, state.At.Add(-recent))
							metadataRecent = clock == "recent"
							deferred := clock == "unknown" && unknownInspections >= observeUnknownHistoryLimit
							if clock == "old" || deferred {
								status := "outside_window"
								if deferred {
									status = "deferred"
								}
								entry := observeLiveAgent{ID: a.ID, MetadataKey: metadataKey, State: status}
								if stampErr == nil {
									copy := stamp
									entry.SourceStamp = &copy
									state.Sources = append(state.Sources, stamp)
								}
								state.Agents = append(state.Agents, entry)
								counts[status]++
								state.WorkerScans["metadata_"+clock]++
								continue agentsLoop
							}
							if clock == "unknown" {
								unknownInspections++
							}
						}
						priorTail := loadObserveTail(old.TailSnapshot, safe, a.ID, workLimit)
						owned, incremental, e := scanObserveTail(safe, a.ID, workLimit, priorTail)
						capturedStamp := owned.Stamp
						sourceStamp = &capturedStamp
						if e != nil {
							stamp.Stable = false
							sourceStamp = &stamp
						}
						if sourceStamp == nil {
							stamp.Stable = false
							sourceStamp = &stamp
						}
						state.Sources = append(state.Sources, *sourceStamp)
						if e != nil {
							acquisitionErr = e
						} else {
							tail = &owned
							tailSnapshot, err = retainObserveTail(owned)
							if err != nil {
								return "", nil, err
							}
							mode := "fresh_stream"
							if incremental {
								mode = "verified_append"
							}
							state.WorkerScans[mode]++
							records = append(records, owned.records()...)
							if owned.Available {
								detail = "available"
							}
							sortObserveRecords(records)
						}
					}
				}
			}
		} else {
			records, _, detail, acquisitionErr = ledgerRecords(canonical, a.ID)
		}
		fingerprint, latest, unknown, e := observeAgentEvidence(a, records, handoffs, detail, task, policies, tail != nil && tail.Owned > 0)
		if e != nil {
			return "", nil, e
		}
		if tail != nil {
			fingerprint = captureHash([]byte(fingerprint + "\x00" + tail.digest()))
			if tail.Latest.After(latest) {
				latest = tail.Latest
			}
			unknown = unknown || tail.Unknown
		}
		fingerprint = captureHash([]byte(fingerprint + "\x00" + source))
		if acquisitionErr != nil {
			fingerprint = captureHash([]byte(fingerprint + acquisitionErr.Error()))
		}
		var oldPtr *observeLiveAgent
		if exists {
			oldPtr = &old
		}
		entry := observeLiveAgent{ID: a.ID, Fingerprint: fingerprint, State: "outside_window", MetadataKey: metadataKey, SourceStamp: sourceStamp, TailSnapshot: tailSnapshot}
		if metadataRecent || observeEligible(oldPtr, fingerprint, latest, unknown, state.At.Add(-recent), previous != "") {
			windowRecords, earlier := observeWorkWindow(records, workLimit)
			signals, evidence := collectAttentionSignals(policies, windowRecords, detail, a.ID, task, source, acquisitionErr)
			for i := range signals {
				if signals[i].Window.Selected > 0 {
					signals[i].Window.Earlier += earlier
				}
			}
			if tail != nil {
				for i := range signals {
					w := &signals[i].Window
					if w.Selected == w.Requested {
						if n, ok := tail.earlier(w.FirstEvent, parentRecords); ok {
							w.Earlier = n
						}
					}
				}
			}
			context := attentionTaskContextFor(a.ID, canonical, records, handoffs)
			if tail != nil && context.Source != "authenticated_launches" {
				context.Total, context.Unavailable = tail.TaskTotal, tail.TaskUnavailable
				context.Omitted = context.Total - len(context.Evidence)
				if context.Omitted > 0 && len(context.Evidence) > 0 {
					context.Status = "partial"
				}
			}
			first := signals[0]
			room := attentionRoom{ID: a.ID, Label: reviewLabel(cleanBundleText(a.Description)), Signals: signals, TaskContext: &context, Metrics: first.Metrics, Window: first.Window, Result: legacyAttentionResult(first)}
			entry.State = "evaluated"
			if detail != "available" {
				entry.State = "unavailable"
			}
			for _, s := range signals {
				outcomes[s.Outcome]++
				if s.Outcome == "error" {
					entry.State = "error"
				}
			}
			entry.Room = &room
			evaluated++
			batch.Page.Rooms = append(batch.Page.Rooms, room)
			batch.Page.SignalEvaluations += len(signals)
			batch.Evidence[a.ID] = evidence
		} else if exists {
			entry = old
			entry.MetadataKey, entry.SourceStamp = metadataKey, sourceStamp
			entry.TailSnapshot = tailSnapshot
			if entry.Room != nil {
				entry.State = "unchanged"
			}
		}
		counts[entry.State]++
		state.Agents = append(state.Agents, entry)
		if len(batch.Page.Rooms) == 20 {
			if err = flush(); err != nil {
				return "", nil, err
			}
		}
	}
	if err = flush(); err != nil {
		return "", nil, err
	}
	removed := 0
	for id := range prior {
		if !known[id] {
			removed++
		}
	}
	var out bytes.Buffer
	fmt.Fprintf(&out, "\nAttention signals — conversation-wide recent-change coverage\nInitial window: %s; unknown recency accounted for. Refresh baseline: %s\nCoverage: population=%d; evaluated=%d; unchanged=%d; outside_window=%d; unavailable=%d; error=%d; deferred=%d; removed_since_prior=%d\n", recent, previous, len(agents), counts["evaluated"], counts["unchanged"], counts["outside_window"], counts["unavailable"], counts["error"], counts["deferred"], removed)
	fmt.Fprintf(&out, "Initial worker acquisition uses source mtime plus attributed parent timestamps, not the parent file mtime. Unknown worker history inspection limit: %d; used: %d. Deferred histories remain uninspected. Source changes are not proof of activity.\n", observeUnknownHistoryLimit, unknownInspections)
	fmt.Fprintf(&out, "Evaluations attempted: %d agents; %d signals. Coverage outcomes: %v\nDetail sample: %d of %d newly evaluated agents; detail omissions do not reduce evaluation coverage.\n", evaluated, evaluated*len(policies), outcomes, displayed, evaluated)
	fmt.Fprintln(&out, "Unchanged results retain their prior attention snapshot and are not re-evaluated, resolved, or healthy. Use retained coverage pages for every agent and its evidence reference.")
	for _, p := range details {
		fmt.Fprintln(&out, "\nBounded current-evaluation detail (not the full coverage tally):")
		if err = renderAttention(&out, p); err != nil {
			return "", nil, err
		}
	}
	attachObserveStamps(state, capture)
	return out.String(), state, nil
}
func renderObserveCoverage(s observeLiveState, page int) (string, error) {
	pages := (len(s.Agents) + 19) / 20
	if pages == 0 {
		pages = 1
	}
	if page < 1 || page > pages {
		return "", fmt.Errorf("observe: coverage page out of range")
	}
	start := (page - 1) * 20
	end := start + 20
	if end > len(s.Agents) {
		end = len(s.Agents)
	}
	var out bytes.Buffer
	fmt.Fprintf(&out, "Retained coverage page %d/%d · population=%d · observed=%s\n", page, pages, len(s.Agents), s.At.Format(time.RFC3339Nano))
	for _, a := range s.Agents[start:end] {
		fmt.Fprintf(&out, "%s · %s · attention snapshot: %s\n", observeLabel(a.ID), a.State, a.Snapshot)
		if a.Room != nil {
			for _, sig := range a.Room.Signals {
				fmt.Fprintf(&out, "  Signal: %s · metric v2 · prior outcome: %s · condition: %s · applicability: %s\n", sig.ID, sig.Outcome, sig.Condition.Status, sig.Applicability.Status)
			}
		}
	}
	if out.Len() > 200000 {
		return "", fmt.Errorf("observe: coverage page exceeds output limit")
	}
	return out.String(), nil
}
