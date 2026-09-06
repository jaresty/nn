package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

// Recognized transcript schema labels. "unknown" is the residual bucket.
const (
	schemaSDKCLI     = "sdk-cli"
	schemaPi         = "pi"
	schemaClaudeCode = "claude-code"
	schemaUnknown    = "unknown"
)

// newTranscriptCmd is the parent for the subagent-transcript navigator (ADR-0042).
// The spine is pure-Go with zero external dependency; DuckDB is escape-hatch-only.
func newTranscriptCmd(_ *rootState) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transcript",
		Short: "Navigate subagent execution transcripts (ADR-0042)",
	}
	cmd.AddCommand(newTranscriptScanCmd(), newTranscriptDoctorCmd(), newTranscriptTreeCmd(), newTranscriptShowCmd())
	return cmd
}

func newTranscriptScanCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "scan [dir]",
		Short: "Discover transcript files and classify each by recognized schema",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			counts, err := scanTranscriptDir(dir)
			if err != nil {
				return err
			}
			// Deterministic order: known schemas first, unknown last.
			order := []string{schemaClaudeCode, schemaSDKCLI, schemaPi, schemaUnknown}
			out := cmd.OutOrStdout()
			for _, schema := range order {
				fmt.Fprintf(out, "%s: %d\n", schema, counts[schema])
			}
			return nil
		},
	}
}

func newTranscriptDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Report duckdb availability (escape-hatch-only; not required for scan)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			if path, err := exec.LookPath("duckdb"); err == nil {
				fmt.Fprintf(out, "duckdb: found at %s\n", path)
			} else {
				fmt.Fprintln(out, "duckdb: not found on PATH")
			}
			fmt.Fprintln(out, "duckdb is escape-hatch-only: it is required only for unknown-schema queries, not for scan or known-schema commands.")
			return nil
		},
	}
}

// scanTranscriptDir walks dir for *.jsonl transcript files (skipping subagent
// child files, which are classified with their parent session) and returns a
// count per recognized schema.
func scanTranscriptDir(dir string) (map[string]int, error) {
	counts := map[string]int{}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		// Subagent child files under a subagents/ dir are part of their parent
		// session's classification, not standalone transcripts.
		if filepath.Base(filepath.Dir(path)) == "subagents" {
			return nil
		}
		counts[classifyTranscript(path)]++
		return nil
	})
	if err != nil {
		return nil, err
	}
	return counts, nil
}

// transcriptRecord captures the union of fields used for schema discrimination.
type transcriptRecord struct {
	Type       string          `json:"type"`
	ParentUUID *string         `json:"parentUuid"`
	UUID       string          `json:"uuid"`
	CustomType string          `json:"customType"`
	Message    json.RawMessage `json:"message"`
}

// classifyTranscript reads a session .jsonl and returns its recognized schema.
//
// Discrimination signatures (ADR-0042 recipes):
//   - sdk-cli:     a sibling subagents/agent-*.jsonl + .meta.json layout exists
//   - pi:          any record has customType "subagents:record", or a pi session
//     header record ({"type":"session","version":N})
//   - claude-code: records carry parentUuid/uuid (the interactive uuid DAG) and
//     no separate subagents/ dir
func classifyTranscript(path string) string {
	// sdk-cli signature is structural: a <session>/subagents/ dir beside the file.
	base := strings.TrimSuffix(path, ".jsonl")
	if info, err := os.Stat(filepath.Join(base, "subagents")); err == nil && info.IsDir() {
		if hasSubagentMeta(filepath.Join(base, "subagents")) {
			return schemaSDKCLI
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return schemaUnknown
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	sawUUIDDag := false
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var rec transcriptRecord
		if json.Unmarshal([]byte(line), &rec) != nil {
			continue
		}
		if rec.CustomType == "subagents:record" {
			return schemaPi
		}
		if rec.Type == "session" {
			// pi session header record.
			return schemaPi
		}
		if rec.ParentUUID != nil || rec.UUID != "" {
			sawUUIDDag = true
		}
	}
	if sawUUIDDag {
		return schemaClaudeCode
	}
	return schemaUnknown
}

// hasSubagentMeta reports whether the subagents dir contains at least one
// agent-*.meta.json sidecar (the sdk-cli spawn-edge marker).
func hasSubagentMeta(subagentsDir string) bool {
	entries, err := os.ReadDir(subagentsDir)
	if err != nil {
		return false
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, n := range names {
		if strings.HasPrefix(n, "agent-") && strings.HasSuffix(n, ".meta.json") {
			return true
		}
	}
	return false
}
