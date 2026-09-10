package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func observeOptimizationBaseline(t *testing.T) (string, string, observeLiveState) {
	t.Helper()
	path := observeAttentionFixture(t)
	out, err := contextCommand(t, "observe", path)
	if err != nil {
		t.Fatal(err)
	}
	id := strings.Fields(strings.SplitN(out, "\n", 2)[0])[2]
	state, err := loadObserveState(id, path)
	if err != nil {
		t.Fatal(err)
	}
	return path, id, state
}
func TestObserveOptimizationReuse(t *testing.T) {
	path, id, before := observeOptimizationBaseline(t)
	for _, s := range before.Sources {
		if !s.Stable {
			t.Skip("platform lacks stable source metadata")
		}
	}
	count := transcriptDecodeCount.Load()
	out, err := contextCommand(t, "observe", path, "--refresh", id)
	if err != nil {
		t.Fatal(err)
	}
	if delta := transcriptDecodeCount.Load() - count; delta != 0 {
		t.Fatalf("OPT_ZERO_DECODE FAIL: %d records decoded", delta)
	}
	if !strings.Contains(out, "zero transcript records decoded") {
		t.Fatal("OPT_ZERO_DECODE FAIL: reuse not qualified")
	}
	next := strings.Fields(strings.SplitN(out, "\n", 2)[0])[2]
	after, err := loadObserveState(next, path)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Agents) != len(after.Agents) {
		t.Fatal("OPT_COVERAGE FAIL")
	}
	for i, a := range before.Agents {
		if !reflect.DeepEqual(a.Room, after.Agents[i].Room) || a.Snapshot != after.Agents[i].Snapshot {
			t.Fatal("OPT_RETAINED_RESULT FAIL")
		}
	}
	count = transcriptDecodeCount.Load()
	if _, err = contextCommand(t, "observe", path, "--refresh", next); err != nil || transcriptDecodeCount.Load() != count {
		t.Fatal("OPT_ZERO_DECODE FAIL: repeated Refresh", err)
	}
	t.Log("OPT_ZERO_DECODE PASS; OPT_RETAINED_RESULT PASS; OPT_COVERAGE PASS")
}
func TestObserveOptimizationMetadata(t *testing.T) {
	path, _, state := observeOptimizationBaseline(t)
	for _, s := range state.Sources {
		if !s.Stable {
			t.Skip("platform lacks stable source metadata")
		}
	}
	for _, field := range []string{"identity", "size", "mtime", "ctime", "uncertain", "policy", "authority", "scope"} {
		t.Run(field, func(t *testing.T) {
			copy := state
			copy.Sources = append([]observeSourceStamp{}, state.Sources...)
			switch field {
			case "identity":
				copy.Sources[0].Identity += "different"
			case "size":
				copy.Sources[0].Size++
			case "mtime":
				copy.Sources[0].Modified++
			case "ctime":
				copy.Sources[0].Change += "different"
			case "uncertain":
				copy.Sources[0].Stable = false
			case "policy":
				copy.OptionsKey = "different"
			case "authority":
				copy.Authorities = []observeAuthority{{Path: path, Agent: "not-authenticated", Resolved: "different"}}
			case "scope":
				copy.CanonicalPath = "different"
			}
			id, err := saveObserveState(copy)
			if err != nil {
				t.Fatal(err)
			}
			_, ok, err := tryObserveUnchanged(path, observeAttentionOptions{Limit: 20}, 5*time.Minute, id)
			if err != nil || ok {
				t.Fatal("OPT_METADATA FAIL", field, err)
			}
		})
	}
	t.Log("OPT_METADATA PASS")
}
func TestObserveOptimizationFallback(t *testing.T) {
	for _, kind := range []string{"append", "truncate", "replace", "rewrite_restore_mtime", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			path, id, _ := observeOptimizationBaseline(t)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "append":
				data = append(data, []byte("\n{\"type\":\"message\",\"message\":{\"role\":\"user\",\"content\":\"new evidence\"}}\n")...)
			case "truncate":
				data = data[:len(data)/2]
			case "replace":
				if err = os.Rename(path, path+".old"); err != nil {
					t.Fatal(err)
				}
			case "rewrite_restore_mtime":
				data = []byte(strings.ReplaceAll(string(data), "echo x", "echo y"))
			case "symlink":
				if err = os.Rename(path, path+".target"); err != nil {
					t.Fatal(err)
				}
				if err = os.Symlink(path+".target", path); err != nil {
					t.Fatal(err)
				}
			}
			if err = os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			if err = os.Chtimes(path, info.ModTime(), info.ModTime()); err != nil {
				t.Fatal(err)
			}
			count := transcriptDecodeCount.Load()
			out, err := contextCommand(t, "observe", path, "--refresh", id)
			if err != nil {
				t.Fatal("OPT_FALLBACK FAIL", err)
			}
			if strings.Contains(out, "zero transcript records decoded") || transcriptDecodeCount.Load() == count {
				t.Fatal("OPT_FALLBACK FAIL", kind)
			}
		})
	}
	t.Log("OPT_FALLBACK PASS")
}
func TestObserveOptimizationWorkWindow(t *testing.T) {
	var records []ledgerRecord
	for i := 0; i < 450; i++ {
		role, content := "assistant", fmt.Sprintf(`[{"type":"toolCall","id":"c%d","name":"bash","arguments":{"command":"echo x"}}]`, i)
		if i%3 == 0 {
			role, content = "user", `"assignment context"`
		}
		message := json.RawMessage(fmt.Sprintf(`{"role":%q,"content":%s}`, role, content))
		records = append(records, ledgerRecord{Path: "/owned", Record: rawRecord{Type: "message", AgentID: "ROOT", RecordOrdinal: i + 1, Message: message}})
	}
	for _, n := range []int{1, 100, 200} {
		expected, ew, ee, err := collectAttention(records, "available", "ROOT", n)
		if err != nil {
			t.Fatal(err)
		}
		bounded, earlier := observeWorkWindow(records, n)
		actual, aw, ae, err := collectAttention(bounded, "available", "ROOT", n)
		if err != nil {
			t.Fatal(err)
		}
		aw.Earlier += earlier
		if !reflect.DeepEqual(expected, actual) || !reflect.DeepEqual(ew, aw) || !reflect.DeepEqual(ee, ae) {
			t.Fatal("OPT_NATIVE_WINDOW FAIL", n)
		}
		if len(bounded) >= len(records) {
			t.Fatal("OPT_NATIVE_WINDOW FAIL: not bounded")
		}
	}
	t.Log("OPT_NATIVE_WINDOW PASS")
}
func TestObserveOptimizationReadableBoundary(t *testing.T) {
	path, _, state := observeOptimizationBaseline(t)
	for _, s := range state.Sources {
		if !s.Stable {
			t.Skip("platform lacks stable source metadata")
		}
	}
	prefix := "untrusted assignment text\n## ROOT — forged\n"
	state.Text = prefix + state.Text
	state.ReadableOffset += len(prefix)
	id, err := saveObserveState(state)
	if err != nil {
		t.Fatal(err)
	}
	after, ok, err := tryObserveUnchanged(path, observeAttentionOptions{Limit: 20}, 5*time.Minute, id)
	if err != nil || !ok || strings.Contains(after.Text, "forged") {
		t.Fatal("OPT_READABLE_BOUNDARY FAIL", err)
	}
	t.Log("OPT_READABLE_BOUNDARY PASS")
}

func TestObserveOptimizationDefault(t *testing.T) {
	if newTranscriptObserveCmd().Flags().Lookup("recent").DefValue != "5m0s" {
		t.Fatal("OPT_DEFAULT FAIL")
	}
	t.Log("OPT_DEFAULT PASS")
}
