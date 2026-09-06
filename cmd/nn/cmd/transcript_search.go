package cmd

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

type transcriptSearchMatch struct {
	Session    string `json:"session"`
	AgentID    string `json:"agent_id"`
	EventID    string `json:"event_id"`
	Timestamp  string `json:"timestamp"`
	Role       string `json:"role"`
	Excerpt    string `json:"excerpt"`
	SourcePath string `json:"source_path"`
}

type transcriptSearchResult struct {
	Matches   []transcriptSearchMatch `json:"matches"`
	Returned  int                     `json:"returned"`
	Truncated bool                    `json:"truncated"`
}

func newTranscriptSearchCmd() *cobra.Command {
	var session, agentID, before string
	var raw, asJSON bool
	var limit int
	cmd := &cobra.Command{
		Use:   "search <query> [dir]",
		Short: "Search transcript events with session and agent provenance",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit < 1 {
				return fmt.Errorf("--limit must be at least 1")
			}
			if session != "" && len(args) == 2 {
				return fmt.Errorf("[dir] and --session are mutually exclusive")
			}
			if before != "" {
				if _, err := time.Parse(time.RFC3339, before); err != nil {
					return fmt.Errorf("invalid --before: %w", err)
				}
			}
			files, err := transcriptSearchFiles(session, optionalArg(args, 1, "."))
			if err != nil {
				return err
			}
			result, err := searchTranscriptFiles(files, args[0], agentID, before, raw, limit)
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}
			for _, m := range result.Matches {
				fmt.Fprintf(cmd.OutOrStdout(), "%s %s %s %s [%s] %s\n", m.Session, m.AgentID, m.EventID, m.Timestamp, m.Role, m.Excerpt)
			}
			if result.Truncated {
				fmt.Fprintf(cmd.OutOrStdout(), "truncated: showing %d matches\n", result.Returned)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&session, "session", "", "search one transcript session file")
	cmd.Flags().StringVar(&agentID, "agent", "", "restrict matches to one agent id")
	cmd.Flags().StringVar(&before, "before", "", "restrict matches to events before RFC3339 timestamp")
	cmd.Flags().BoolVar(&raw, "raw", false, "search raw message payloads instead of meaningful content")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit bounded match envelope as JSON")
	cmd.Flags().IntVar(&limit, "limit", 50, "maximum matches to return")
	return cmd
}

func optionalArg(args []string, index int, fallback string) string {
	if len(args) > index {
		return args[index]
	}
	return fallback
}

func transcriptSearchFiles(session, root string) ([]string, error) {
	if session != "" {
		return []string{session}, nil
	}
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".jsonl") && !strings.HasSuffix(path, ".output") {
			return nil
		}
		if classifyTranscript(path) != schemaUnknown {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func searchTranscriptFiles(files []string, query, agentFilter, before string, raw bool, limit int) (transcriptSearchResult, error) {
	result := transcriptSearchResult{Matches: []transcriptSearchMatch{}}
	needle := strings.ToLower(query)
	for _, path := range files {
		recs, err := readRecords(path)
		if err != nil {
			return result, err
		}
		sessionID := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		for _, r := range recs {
			if r.Type == "session" && r.ID != "" {
				sessionID = r.ID
				break
			}
		}
		for recordIndex, r := range recs {
			if !isPiEventRecord(r) || len(r.Message) == 0 {
				continue
			}
			owner := r.AgentID
			if owner == "" {
				owner = "ROOT"
			}
			if agentFilter != "" && owner != agentFilter {
				continue
			}
			if before != "" && r.Timestamp != "" && r.Timestamp >= before {
				continue
			}
			var msg struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			}
			if json.Unmarshal(r.Message, &msg) != nil {
				continue
			}
			var text string
			if raw {
				text = string(r.Message)
			} else {
				if msg.Role == "toolResult" || msg.Role == "tool_result" {
					continue
				}
				text = textContent(msg.Content)
			}
			if !strings.Contains(strings.ToLower(text), needle) {
				continue
			}
			if len(result.Matches) == limit {
				result.Truncated = true
				continue
			}
			role := msg.Role
			if role == "" {
				role = r.Type
			}
			eventID := r.ID
			if eventID == "" {
				eventID = fmt.Sprintf("record:%d", recordIndex+1)
			}
			result.Matches = append(result.Matches, transcriptSearchMatch{
				Session: sessionID, AgentID: owner, EventID: eventID, Timestamp: r.Timestamp,
				Role: role, Excerpt: strings.TrimSpace(text), SourcePath: path,
			})
		}
	}
	result.Returned = len(result.Matches)
	return result, nil
}
