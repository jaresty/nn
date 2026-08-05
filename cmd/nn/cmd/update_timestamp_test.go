package cmd

import (
	"strings"
	"testing"
	"time"
)

// property [1]: nn update output uses RFC3339Nano precision for modified timestamp
func TestUpdateOutputsRFC3339NanoTimestamp(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("new", "--title", "Timestamp test", "--type", "observation", "--content", "body", "--no-edit")
	if err != nil {
		t.Fatalf("nn new: %v", err)
	}
	id := strings.Fields(strings.TrimSpace(out))[1]

	since := noteModified(t, execute, id)

	updateOut, err := execute("update", id, "--title", "Timestamp test updated", "--since", since, "--no-edit")
	if err != nil {
		t.Fatalf("nn update: %v", err)
	}

	// Extract the modified: line
	var modLine string
	for _, line := range strings.Split(updateOut, "\n") {
		if strings.HasPrefix(line, "modified:") {
			modLine = strings.TrimPrefix(line, "modified: ")
			break
		}
	}
	if modLine == "" {
		t.Fatalf("no modified: line in output:\n%s", updateOut)
	}

	// Must parse as RFC3339Nano
	_, err = time.Parse(time.RFC3339Nano, modLine)
	if err != nil {
		t.Errorf("modified timestamp %q does not parse as RFC3339Nano: %v", modLine, err)
	}

	// Must NOT be truncated to seconds (RFC3339 without nanoseconds parses fine with RFC3339Nano,
	// but we can detect second-precision by checking there are no sub-second digits)
	if !strings.Contains(modLine, ".") {
		t.Errorf("modified timestamp %q lacks sub-second precision — expected RFC3339Nano with fractional seconds", modLine)
	}
}
