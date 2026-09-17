package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// repoPath resolves a repo-relative path from the cmd/nn/cmd test package.
func repoPath(t *testing.T, rel ...string) string {
	t.Helper()
	parts := append([]string{"..", "..", ".."}, rel...)
	p, err := filepath.Abs(filepath.Join(parts...))
	if err != nil {
		t.Fatalf("resolve %v: %v", rel, err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("not found at %s: %v", p, err)
	}
	return p
}

// hookStdout mirrors the JSON contract Claude Code parses from a hook's stdout.
type hookStdout struct {
	HookSpecificOutput struct {
		HookEventName     string `json:"hookEventName"`
		AdditionalContext string `json:"additionalContext"`
	} `json:"hookSpecificOutput"`
}

// TestProtocolsReminderEmitsValidJSON drives protocols-reminder.sh down every path it
// can take and asserts the stdout handed to Claude Code parses as JSON. Content
// assertions guard against an escaping fix that buys validity by dropping characters.
func TestProtocolsReminderEmitsValidJSON(t *testing.T) {
	script := repoPath(t, "plugins", "nn-hooks", "scripts", "protocols-reminder.sh")

	cases := []struct {
		name        string
		session     string
		sentinel    bool
		turnCount   string
		wantContain []string
	}{
		{
			name:        "sentinel absent",
			session:     "sess-absent",
			wantContain: []string{"`nn show --global`", "## Protocols", "Permitted invocation form"},
		},
		{
			name:        "sentinel present",
			session:     "sess-present",
			sentinel:    true,
			wantContain: []string{"`nn show --global`", "## Protocols"},
		},
		{
			name:        "nudge threshold turn",
			session:     "sess-nudge",
			turnCount:   "11",
			wantContain: []string{"`nn show --global`", "/nn-session-debrief --partial"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			cacheDir := filepath.Join(home, ".cache", "nn")
			if err := os.MkdirAll(cacheDir, 0o755); err != nil {
				t.Fatal(err)
			}
			if tc.sentinel {
				sentinel := filepath.Join(cacheDir, "global-loaded-"+tc.session)
				if err := os.WriteFile(sentinel, nil, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if tc.turnCount != "" {
				counter := filepath.Join(cacheDir, "turn-count-"+tc.session)
				if err := os.WriteFile(counter, []byte(tc.turnCount), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			var env []string
			for _, kv := range os.Environ() {
				if strings.HasPrefix(kv, "HOME=") {
					continue
				}
				env = append(env, kv)
			}
			env = append(env, "HOME="+home)

			cmd := exec.Command("bash", script)
			cmd.Env = env
			cmd.Stdin = strings.NewReader(`{"session_id":"` + tc.session + `"}`)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("script exited non-zero: %v\nstderr: %s", err, stderr.String())
			}

			var out hookStdout
			if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
				t.Fatalf("stdout is not valid JSON: %v\nstdout: %s", err, stdout.String())
			}
			if out.HookSpecificOutput.HookEventName != "UserPromptSubmit" {
				t.Errorf("hookEventName = %q, want UserPromptSubmit", out.HookSpecificOutput.HookEventName)
			}
			ctx := out.HookSpecificOutput.AdditionalContext
			if ctx == "" {
				t.Fatal("additionalContext is empty")
			}
			for _, want := range tc.wantContain {
				if !strings.Contains(ctx, want) {
					t.Errorf("additionalContext missing %q\ngot:\n%s", want, ctx)
				}
			}
		})
	}
}
