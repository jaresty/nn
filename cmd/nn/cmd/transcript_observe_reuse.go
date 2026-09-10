package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/jaresty/nn/internal/attention"
)

// Metadata is a conservative reuse check, not cryptographic proof of immutable
// content. Filesystems without file identity and change-time support fall back.
type observeSourceStamp struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Modified int64  `json:"modified_ns"`
	Identity string `json:"identity"`
	Change   string `json:"change_time"`
	Stable   bool   `json:"stable"`
}
type observeAuthority struct {
	Path     string `json:"path"`
	Agent    string `json:"agent"`
	Resolved string `json:"resolved"`
}

func stampObserveSource(path string, info os.FileInfo) observeSourceStamp {
	s := observeSourceStamp{Path: path, Size: info.Size(), Modified: info.ModTime().UnixNano()}
	v := reflect.ValueOf(info.Sys())
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return s
	}
	dev, ino := v.FieldByName("Dev"), v.FieldByName("Ino")
	if !dev.IsValid() || !ino.IsValid() {
		return s
	}
	s.Identity = fmt.Sprintf("%v:%v", dev.Interface(), ino.Interface())
	for _, name := range []string{"Ctimespec", "Ctim"} {
		if field := v.FieldByName(name); field.IsValid() {
			s.Change = fmt.Sprint(field.Interface())
			break
		}
	}
	s.Stable = info.Mode().IsRegular() && s.Change != ""
	return s
}
func currentObserveStamp(path string) (observeSourceStamp, error) {
	info, err := os.Stat(path)
	if err != nil {
		return observeSourceStamp{}, err
	}
	return stampObserveSource(path, info), nil
}
func sameObserveStamp(a, b observeSourceStamp) bool { return a.Stable && b.Stable && a == b }
func observeOptionsKey(o observeAttentionOptions, recent time.Duration) (string, error) {
	p, err := attention.Builtins()
	if err != nil {
		return "", err
	}
	tasks, err := parseAgentTasks(o.AgentTasks)
	if err != nil {
		return "", err
	}
	b, err := json.Marshal([]any{"observe-reuse-v1", p, o.Task, tasks, recent, o.Limit})
	return captureHash(b), err
}
func attachObserveStamps(s *observeLiveState, c *transcriptCapture) {
	if c == nil {
		return
	}
	for _, source := range c.Sources {
		if source.ObservationStamp != nil {
			s.Sources = append(s.Sources, *source.ObservationStamp)
		}
	}
	sort.Slice(s.Sources, func(i, j int) bool { return s.Sources[i].Path < s.Sources[j].Path })
	for _, loc := range piBackgroundLocators(c.Sources[c.Path].Records) {
		s.Authorities = append(s.Authorities, observeAuthority{loc.Path, loc.AgentID, c.resolve(loc.Path, loc.AgentID)})
	}
	sort.Slice(s.Authorities, func(i, j int) bool {
		if s.Authorities[i].Path != s.Authorities[j].Path {
			return s.Authorities[i].Path < s.Authorities[j].Path
		}
		return s.Authorities[i].Agent < s.Authorities[j].Agent
	})
}
func tryObserveUnchanged(session string, o observeAttentionOptions, recent time.Duration, previous string) (*observeLiveState, bool, error) {
	old, err := loadObserveState(previous, session)
	if err != nil {
		return nil, false, err
	}
	key, err := observeOptionsKey(o, recent)
	if err != nil {
		return nil, false, err
	}
	if old.OptionsKey != key || len(old.Sources) == 0 {
		return nil, false, nil
	}
	canonical, scopeErr := contextPath(session)
	if scopeErr != nil || canonical != old.CanonicalPath {
		return nil, false, nil
	}
	for _, a := range old.Authorities {
		if validatePiSidechainPath(a.Path, a.Agent) != a.Resolved {
			return nil, false, nil
		}
	}
	for _, stamp := range old.Sources {
		now, e := currentObserveStamp(stamp.Path)
		if e != nil || !sameObserveStamp(stamp, now) {
			return nil, false, nil
		}
	}
	cut := old.ReadableOffset
	if cut <= 0 || cut >= len(old.Text) || !strings.HasPrefix(old.Text[cut:], "\n## ROOT —") {
		return nil, false, nil
	}
	old.At = time.Now().UTC()
	old.WorkerScans = nil
	counts := map[string]int{}
	outcomes := map[string]int{}
	for i := range old.Agents {
		a := &old.Agents[i]
		if a.Room != nil {
			a.State = "unchanged"
			for _, sig := range a.Room.Signals {
				outcomes[sig.Outcome]++
			}
		}
		counts[a.State]++
	}
	prefix := fmt.Sprintf("Observation: %s\nMetadata-checked unchanged Refresh: zero transcript records decoded. Readable samples and signal evidence retained, not reacquired.\nCoverage: population=%d; evaluated=0; unchanged=%d; outside_window=%d; unavailable=0; error=0; deferred=0\nRetained prior signal outcomes (not re-evaluated): %v\nNo resolution, health or current-activity inference. Metadata continuity is not cryptographic content verification.\n", observeLabel(session), len(old.Agents), counts["unchanged"], counts["outside_window"], outcomes)
	old.Text = prefix + old.Text[cut:]
	old.ReadableOffset = len(prefix)
	return &old, true, nil
}
