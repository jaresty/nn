package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"
)

func TestObserveMetadataRecentCandidates(t *testing.T) {
	for _, mode := range []string{"source", "parent"} {
		t.Run(mode, func(t *testing.T) {
			path := recencyFixture(t, 1, true)
			records, e := readRecords(path)
			if e != nil {
				t.Fatal(e)
			}
			loc := piBackgroundLocators(records)[0]
			if mode == "source" {
				now := time.Now()
				if e = os.Chtimes(loc.Path, now, now); e != nil {
					t.Fatal(e)
				}
			} else {
				b, e := os.ReadFile(path)
				if e != nil {
					t.Fatal(e)
				}
				lines := strings.Split(strings.TrimSpace(string(b)), "\n")
				for i, line := range lines {
					var row map[string]any
					if e = json.Unmarshal([]byte(line), &row); e != nil {
						t.Fatal(e)
					}
					if row["id"] == "launch-worker-000" {
						row["timestamp"] = time.Now().UTC().Format(time.RFC3339Nano)
						b, e = json.Marshal(row)
						if e != nil {
							t.Fatal(e)
						}
						lines[i] = string(b)
					}
				}
				if e = os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0600); e != nil {
					t.Fatal(e)
				}
			}
			_, state, e := observeLiveSection(path, observeAttentionOptions{Limit: 1}, 5*time.Minute, "")
			if e != nil {
				t.Fatal(e)
			}
			for _, a := range state.Agents {
				if a.ID == "worker-000" {
					if a.Room == nil || a.Room.Window.Selected != 1 {
						t.Fatal("METADATA_CANDIDATE FAIL", a)
					}
					return
				}
			}
			t.Fatal("candidate absent")
		})
	}
}
