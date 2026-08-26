package note

import (
	"strings"
	"testing"
)

func TestFollowsIsCanonicalProcessHistoryLinkType(t *testing.T) {
	if !IsKnownLinkType("follows") {
		t.Fatal("follows is not accepted as a canonical link type")
	}
	description := LinkTypeDescriptions["follows"]
	for _, boundary := range []string{"later workflow or inquiry step", "without implying derivation", "evidential dependence", "conceptual extension", "task dependency"} {
		if !strings.Contains(description, boundary) {
			t.Errorf("follows description missing semantic boundary %q: %q", boundary, description)
		}
	}
}
