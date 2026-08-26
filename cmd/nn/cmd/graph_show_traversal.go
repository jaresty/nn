package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jaresty/nn/internal/note"
)

type graphShowTraversalOptions struct {
	direction      string
	linkTypes      map[string]bool
	statuses       map[note.Status]bool
	representation string
}

func parseGraphTraversalCSV(command, value, flag string) ([]string, error) {
	if value == "" {
		return nil, nil
	}
	var values []string
	seen := make(map[string]bool)
	for _, raw := range strings.Split(value, ",") {
		item := strings.TrimSpace(raw)
		if item == "" {
			return nil, fmt.Errorf("%s: --%s contains an empty value", command, flag)
		}
		if !seen[item] {
			seen[item] = true
			values = append(values, item)
		}
	}
	return values, nil
}

func newGraphTraversalOptions(command, direction, links, statuses, representation string) (graphShowTraversalOptions, error) {
	opts := graphShowTraversalOptions{direction: direction, representation: representation}
	if direction != "outgoing" && direction != "incoming" && direction != "both" {
		return opts, fmt.Errorf("%s: invalid --direction %q (want outgoing, incoming, or both)", command, direction)
	}
	linkValues, err := parseGraphTraversalCSV(command, links, "links")
	if err != nil {
		return opts, err
	}
	if len(linkValues) > 0 {
		opts.linkTypes = make(map[string]bool, len(linkValues))
		for _, linkType := range linkValues {
			if !note.IsKnownLinkType(linkType) || linkType == "" {
				return opts, fmt.Errorf("%s: invalid --links value %q", command, linkType)
			}
			opts.linkTypes[linkType] = true
		}
	}
	statusValues, err := parseGraphTraversalCSV(command, statuses, "status")
	if err != nil {
		return opts, err
	}
	if len(statusValues) > 0 {
		opts.statuses = make(map[note.Status]bool, len(statusValues))
		for _, value := range statusValues {
			status := note.Status(value)
			if !status.IsValid() {
				return opts, fmt.Errorf("%s: invalid --status value %q (want draft, reviewed, or permanent)", command, value)
			}
			opts.statuses[status] = true
		}
	}
	return opts, nil
}

func newGraphShowTraversalOptions(direction, links, statuses, representation string) (graphShowTraversalOptions, error) {
	return newGraphTraversalOptions("graph show", direction, links, statuses, representation)
}

func (opts graphShowTraversalOptions) allowsLink(linkType string) bool {
	return len(opts.linkTypes) == 0 || opts.linkTypes[linkType]
}

func (opts graphShowTraversalOptions) allowsNote(n *note.Note) bool {
	if len(opts.statuses) > 0 && !opts.statuses[n.Status] {
		return false
	}
	return opts.representation == "" || n.Representation == opts.representation
}

type graphShowNeighbor struct {
	n    *note.Note
	link note.Link
}

func graphShowBFS(root *note.Note, byID map[string]*note.Note, maxDepth int, opts graphShowTraversalOptions) []depthEntry {
	var inbound map[string][]graphShowNeighbor
	if opts.direction != "outgoing" {
		inbound = make(map[string][]graphShowNeighbor)
		ids := make([]string, 0, len(byID))
		for id := range byID {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			source := byID[id]
			for _, lnk := range source.Links {
				if target := byID[lnk.TargetID]; target != nil {
					inbound[target.ID] = append(inbound[target.ID], graphShowNeighbor{source, lnk})
				}
			}
		}
	}

	visited := map[string]bool{root.ID: true}
	queue := []depthEntry{{n: root, level: 0}}
	var ordered []depthEntry
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		ordered = append(ordered, cur)
		if cur.level >= maxDepth {
			continue
		}
		var neighbors []graphShowNeighbor
		if opts.direction != "incoming" {
			for _, lnk := range cur.n.Links {
				if target := byID[lnk.TargetID]; target != nil {
					neighbors = append(neighbors, graphShowNeighbor{target, lnk})
				}
			}
		}
		if opts.direction != "outgoing" {
			neighbors = append(neighbors, inbound[cur.n.ID]...)
		}
		for _, neighbor := range neighbors {
			if visited[neighbor.n.ID] || !opts.allowsLink(neighbor.link.Type) || !opts.allowsNote(neighbor.n) {
				continue
			}
			visited[neighbor.n.ID] = true
			queue = append(queue, depthEntry{n: neighbor.n, level: cur.level + 1})
		}
	}
	return ordered
}
