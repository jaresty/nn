package cmd

import (
	"encoding/json"
	"testing"
)

// property [1]: applies_when is propagated to created note when present in spec
// property [2]: applies_when is empty when absent from spec
// property [3]: JSON with applies_when key is accepted without error
func TestBulkNewAppliesWhen(t *testing.T) {
	_, execute := setupNotebook(t)

	input := `[
		{"title": "Protocol Note", "type": "protocol", "content": "body", "applies_when": "before every deploy"},
		{"title": "Regular Note", "type": "concept", "content": "body2"}
	]`
	out, err := execute("bulk-new", "--json", input)
	if err != nil {
		t.Fatalf("nn bulk-new: %v\noutput: %s", err, out)
	}

	listOut, err := execute("list", "--json", "--fields", "id,title,applies_when")
	if err != nil {
		t.Fatalf("nn list: %v", err)
	}
	var notes []map[string]any
	if err := json.Unmarshal([]byte(listOut), &notes); err != nil {
		t.Fatalf("parse list JSON: %v\n%s", err, listOut)
	}

	for _, n := range notes {
		title := n["title"].(string)
		switch title {
		case "Protocol Note":
			// property [1]
			if n["applies_when"] != "before every deploy" {
				t.Errorf("Protocol Note: expected applies_when='before every deploy', got %v", n["applies_when"])
			}
		case "Regular Note":
			// property [2]
			if v, ok := n["applies_when"]; ok && v != "" && v != nil {
				t.Errorf("Regular Note: expected applies_when empty, got %v", v)
			}
		}
	}
}
