package cmd

import (
	"sort"

	"github.com/jaresty/nn/internal/note"
)

type labelPropagationResult struct {
	groups    map[string][]*note.Note
	outDegree map[string]int
	inDegree  map[string]int
}

// labelPropagationClusters applies the clusters command's deterministic algorithm
// to the complete supplied graph. Search projections must happen after this step.
func labelPropagationClusters(notes []*note.Note) labelPropagationResult {
	adj := make(map[string][]string, len(notes))
	outDegree := make(map[string]int, len(notes))
	inDegree := make(map[string]int, len(notes))
	for _, n := range notes {
		for _, lnk := range n.Links {
			adj[n.ID] = append(adj[n.ID], lnk.TargetID)
			adj[lnk.TargetID] = append(adj[lnk.TargetID], n.ID)
			outDegree[n.ID]++
			inDegree[lnk.TargetID]++
		}
	}

	labels := make(map[string]string, len(notes))
	ids := make([]string, 0, len(notes))
	for _, n := range notes {
		labels[n.ID] = n.ID
		ids = append(ids, n.ID)
	}
	sort.Strings(ids)

	for iter := 0; iter < 20; iter++ {
		changed := false
		for _, id := range ids {
			neighbors := adj[id]
			if len(neighbors) == 0 {
				continue
			}
			freq := make(map[string]int)
			for _, neighbor := range neighbors {
				freq[labels[neighbor]]++
			}
			bestLabel := labels[id]
			bestCount := 0
			for label, count := range freq {
				if count > bestCount || (count == bestCount && label < bestLabel) {
					bestLabel = label
					bestCount = count
				}
			}
			if bestLabel != labels[id] {
				labels[id] = bestLabel
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	groups := make(map[string][]*note.Note)
	for _, n := range notes {
		groups[labels[n.ID]] = append(groups[labels[n.ID]], n)
	}
	return labelPropagationResult{groups: groups, outDegree: outDegree, inDegree: inDegree}
}

func labelPropagationRepresentative(members []*note.Note, outDegree, inDegree map[string]int) *note.Note {
	var representative *note.Note
	for _, n := range members {
		degree := outDegree[n.ID] + inDegree[n.ID]
		if representative == nil || degree > outDegree[representative.ID]+inDegree[representative.ID] ||
			(degree == outDegree[representative.ID]+inDegree[representative.ID] && n.ID < representative.ID) {
			representative = n
		}
	}
	return representative
}

type fullGraphRegion struct {
	representative *note.Note
	size           int
}

func fullGraphRegionIndex(notes []*note.Note) map[string]*fullGraphRegion {
	clustering := labelPropagationClusters(notes)
	regionsByNote := make(map[string]*fullGraphRegion, len(notes))
	for _, members := range clustering.groups {
		region := &fullGraphRegion{
			representative: labelPropagationRepresentative(members, clustering.outDegree, clustering.inDegree),
			size:           len(members),
		}
		for _, member := range members {
			regionsByNote[member.ID] = region
		}
	}
	return regionsByNote
}
