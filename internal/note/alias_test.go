package note

import (
	"strings"
	"testing"

	"github.com/kljensen/snowball/english"
)

// contains reports whether want is an element of got.
func contains(got []string, want string) bool {
	for _, g := range got {
		if g == want {
			return true
		}
	}
	return false
}

// Property 1: T[lower(a)] = baseTokenize(N.Title).
// The alias table maps each note's lowercased alias to that note's
// base-tokenized (unstemmed-relative-to-the-map, i.e. tokenize's own output
// minus alias expansion) title tokens.
func TestBuildAliases_MapsAliasToTitleTokens(t *testing.T) {
	n := &Note{ID: "n1", Title: "Test-driven development", Aliases: []string{"TDD"}}
	table, err := BuildAliases([]*Note{n})
	if err != nil {
		t.Fatalf("BuildAliases returned unexpected error: %v", err)
	}
	got, ok := table["tdd"]
	if !ok {
		t.Fatalf("property 1: alias key %q absent from table %v", "tdd", table)
	}
	// baseTokenize(N.Title): the title's own tokens. "test", "driven",
	// "development" after stemming become "test", "driven", "develop".
	want := tokenize("Test-driven development")
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("property 1: table[%q] = %v, want %v", "tdd", got, want)
	}
}

// Property 2': after SetAliases(BuildAliases(corpus)), tokenize(s) includes the
// stemmed expansion for any alias token in s.
func TestTokenize_ExpandsInstalledAlias(t *testing.T) {
	n := &Note{ID: "n1", Title: "Test-driven development", Aliases: []string{"TDD"}}
	table, err := BuildAliases([]*Note{n})
	if err != nil {
		t.Fatalf("BuildAliases error: %v", err)
	}
	SetAliases(table)
	defer SetAliases(nil) // restore for isolation

	got := tokenize("TDD")
	for _, w := range []string{"test", "driven", "develop"} {
		if !contains(got, w) {
			t.Fatalf("property 2': tokenize(%q) = %v, missing expansion token %q", "TDD", got, w)
		}
	}
}

// Property 2' (pre-stem ordering component): expansion tokens are themselves
// stemmed, i.e. the raw unstemmed title word does not leak into the output when
// its stem differs.
func TestTokenize_ExpandsBeforeStemming(t *testing.T) {
	n := &Note{ID: "n1", Title: "Development practices", Aliases: []string{"DEVX"}}
	table, err := BuildAliases([]*Note{n})
	if err != nil {
		t.Fatalf("BuildAliases error: %v", err)
	}
	SetAliases(table)
	defer SetAliases(nil)

	got := tokenize("DEVX")
	// "development" stems to "develop"; the stemmed form must be present.
	if !contains(got, english.Stem("development", false)) {
		t.Fatalf("property 2' pre-stem: tokenize(%q) = %v, missing stemmed %q", "DEVX", got, english.Stem("development", false))
	}
	// The raw unstemmed "development" must NOT appear (proves stemming ran on expansion).
	if contains(got, "development") && english.Stem("development", false) != "development" {
		t.Fatalf("property 2' pre-stem: raw unstemmed token leaked: %v", got)
	}
}

// Property 5a: BuildAliases errors on a duplicate case-folded alias key.
func TestBuildAliases_ErrorsOnDuplicateKey(t *testing.T) {
	n1 := &Note{ID: "n1", Title: "Test-driven development", Aliases: []string{"TDD"}}
	n2 := &Note{ID: "n2", Title: "Type-driven design", Aliases: []string{"tdd"}}
	_, err := BuildAliases([]*Note{n1, n2})
	if err == nil {
		t.Fatalf("property 5a: expected error on duplicate case-folded alias key, got nil")
	}
}

// Property 5b: the duplicate-key error names both declaring note IDs.
func TestBuildAliases_DuplicateErrorNamesBothIDs(t *testing.T) {
	n1 := &Note{ID: "n1", Title: "Test-driven development", Aliases: []string{"TDD"}}
	n2 := &Note{ID: "n2", Title: "Type-driven design", Aliases: []string{"tdd"}}
	_, err := BuildAliases([]*Note{n1, n2})
	if err == nil {
		t.Fatalf("property 5b: expected error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "n1") || !strings.Contains(msg, "n2") {
		t.Fatalf("property 5b: error %q does not name both note IDs n1 and n2", msg)
	}
}

// Property 6a + 6b: aliases round-trip through Marshal then Parse.
func TestNote_AliasesRoundTrip(t *testing.T) {
	n := &Note{
		ID:      "20260101000000-0001",
		Title:   "Test-driven development",
		Type:    TypeConcept,
		Status:  StatusPermanent,
		Aliases: []string{"TDD", "TDDev"},
		Body:    "body",
	}
	data, err := n.Marshal()
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	got, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if strings.Join(got.Aliases, ",") != strings.Join(n.Aliases, ",") {
		t.Fatalf("property 6: round-trip aliases = %v, want %v", got.Aliases, n.Aliases)
	}
}

// Adequacy discriminator for property 1: a distinguishing state where title and
// body tokens differ, so an implementation keying the table off body tokens
// (a surrogate) would produce a different result than keying off the title.
// R_P(S): the table must equal title tokens. R_A(S): the guard reads the table.
func TestBuildAliases_UsesTitleNotBody(t *testing.T) {
	n := &Note{ID: "n1", Title: "Alpha", Aliases: []string{"AK"}, Body: "beta gamma delta"}
	table, err := BuildAliases([]*Note{n})
	if err != nil {
		t.Fatalf("BuildAliases error: %v", err)
	}
	got := table["ak"]
	// Must be title tokens ("alpha"), not body tokens ("beta","gamma","delta").
	if strings.Join(got, ",") != "alpha" {
		t.Fatalf("adequacy property 1: table[ak] = %v, want [alpha] (title, not body)", got)
	}
	if contains(got, "beta") {
		t.Fatalf("adequacy property 1: table used body tokens: %v", got)
	}
}
