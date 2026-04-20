package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

// Assertion: TestCaptureCommandExists — command is registered and runs without error.
func TestCaptureCommandExists(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("capture", "--title", "Test Capture", "--content", "some raw content")
	if err != nil {
		t.Fatalf("capture command failed: %v", err)
	}
}

// Assertion: TestCaptureCreatesNote — note appears in nn list after capture.
func TestCaptureCreatesNote(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("capture", "--title", "My Observation", "--content", "some raw content to capture")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	out, err := execute("list", "--json")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, "My Observation") {
		t.Errorf("expected captured note in list; got:\n%s", out)
	}
}

// Assertion: TestCaptureDefaultType — default type is observation.
func TestCaptureDefaultType(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("capture", "--title", "Default Type Note", "--content", "content")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	out, err := execute("list", "--json")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var notes []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &notes); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	var found bool
	for _, n := range notes {
		if n["title"] == "Default Type Note" {
			found = true
			if n["type"] != "observation" {
				t.Errorf("expected default type 'observation'; got %q", n["type"])
			}
		}
	}
	if !found {
		t.Errorf("captured note not found in list")
	}
}

// Assertion: TestCaptureExplicitType — --type concept overrides the default.
func TestCaptureExplicitType(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("capture", "--title", "Concept Note", "--content", "content", "--type", "concept")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	out, err := execute("list", "--json")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var notes []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &notes); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	for _, n := range notes {
		if n["title"] == "Concept Note" {
			if n["type"] != "concept" {
				t.Errorf("expected type 'concept'; got %q", n["type"])
			}
			return
		}
	}
	t.Errorf("captured note not found in list")
}

// Assertion: TestCaptureDefaultStatus — captured note has draft status.
func TestCaptureDefaultStatus(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("capture", "--title", "Draft Note", "--content", "content")
	if err != nil {
		t.Fatalf("capture: %v", err)
	}
	out, err := execute("list", "--json")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	var notes []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &notes); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	for _, n := range notes {
		if n["title"] == "Draft Note" {
			if n["status"] != "draft" {
				t.Errorf("expected status 'draft'; got %q", n["status"])
			}
			return
		}
	}
	t.Errorf("captured note not found in list")
}
