package rules

import (
	"strings"

	"github.com/jaresty/nn/internal/note"
)

// FactsFromNotes derives the closed-world fact base from a set of notes. Every
// note automatically exposes these predicates (per ADR-0019):
//
//	note(ID, Type, Status)
//	link(From, To, LinkType)
//	tag(ID, Tag)
//	open_item(ID, Text)          — one per unchecked "- [ ]" line
//	expires(ID, YYYY-MM-DD)       — only if the note has an expiry
//	representation(ID, Rep)       — only if the note has a representation
func FactsFromNotes(notes []*note.Note) []Fact {
	var facts []Fact
	for _, n := range notes {
		facts = append(facts, Fact{Pred: "note", Args: []string{n.ID, string(n.Type), string(n.Status)}})
		for _, lnk := range n.Links {
			facts = append(facts, Fact{Pred: "link", Args: []string{n.ID, lnk.TargetID, lnk.Type}})
		}
		for _, tag := range n.Tags {
			facts = append(facts, Fact{Pred: "tag", Args: []string{n.ID, tag}})
		}
		if n.Representation != "" {
			facts = append(facts, Fact{Pred: "representation", Args: []string{n.ID, n.Representation}})
		}
		if n.Expires != nil {
			facts = append(facts, Fact{Pred: "expires", Args: []string{n.ID, n.Expires.Format("2006-01-02")}})
		}
		for _, text := range openItems(n.Body) {
			facts = append(facts, Fact{Pred: "open_item", Args: []string{n.ID, text}})
		}
	}
	return facts
}

// openItems returns the trimmed text of each unchecked "- [ ]" checkbox line.
func openItems(body string) []string {
	var out []string
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "- [ ]" {
			out = append(out, "")
			continue
		}
		if strings.HasPrefix(trimmed, "- [ ] ") {
			out = append(out, strings.TrimSpace(trimmed[len("- [ ] "):]))
		}
	}
	return out
}
