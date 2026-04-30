package cmd

import (
	"fmt"
	"io"
	"sort"

	"github.com/jaresty/nn/internal/note"
)

// printSuggestions prints advisory link and tag suggestions to w after a note write.
// Failures are silently ignored — suggestions are advisory only.
func printSuggestions(w io.Writer, state *rootState, n *note.Note) {
	notes, err := state.backend.List()
	if err != nil {
		return
	}

	query := n.Title + " " + n.Body

	// Candidate notes: exclude the new note itself.
	var others []*note.Note
	for _, o := range notes {
		if o.ID != n.ID {
			others = append(others, o)
		}
	}
	if len(others) == 0 {
		return
	}

	scores := note.BM25Scores(others, query, nil)

	// Top similar notes (non-zero score, cap at 5).
	var similar []*note.Note
	for _, o := range others {
		if scores[o.ID] > 0 {
			similar = append(similar, o)
		}
	}
	sort.SliceStable(similar, func(i, j int) bool {
		return scores[similar[i].ID] > scores[similar[j].ID]
	})
	if len(similar) > 5 {
		similar = similar[:5]
	}

	// Link suggestions: top similar notes not already linked.
	linkedIDs := make(map[string]bool)
	for _, lnk := range n.Links {
		linkedIDs[lnk.TargetID] = true
	}
	var linkSuggestions []string
	for _, o := range similar {
		if !linkedIDs[o.ID] {
			linkSuggestions = append(linkSuggestions, fmt.Sprintf("%s %q (%.2f)", o.ID, o.Title, scores[o.ID]))
		}
	}

	// Tag suggestions: tags from ≥2 similar notes that n lacks.
	focalTags := make(map[string]bool)
	for _, t := range n.Tags {
		focalTags[t] = true
	}
	tagCount := map[string]int{}
	for _, o := range similar {
		for _, t := range o.Tags {
			if !focalTags[t] {
				tagCount[t]++
			}
		}
	}
	var tagSuggestions []string
	for t, count := range tagCount {
		if count >= 2 {
			tagSuggestions = append(tagSuggestions, t)
		}
	}
	sort.Strings(tagSuggestions)

	if len(linkSuggestions) == 0 && len(tagSuggestions) == 0 {
		return
	}

	fmt.Fprintln(w, "\nSuggestions:")
	if len(linkSuggestions) > 0 {
		fmt.Fprintf(w, "  links: %s\n", joinStrings(linkSuggestions, ", "))
	}
	if len(tagSuggestions) > 0 {
		fmt.Fprintf(w, "  tags:  %s\n", joinStrings(tagSuggestions, ", "))
	}
}

func joinStrings(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
