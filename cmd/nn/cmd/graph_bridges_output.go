package cmd

import (
	"fmt"
	"io"
	"sort"

	"github.com/jaresty/nn/internal/note"
)

type bridgeCandidate struct {
	id             string
	title          string
	score          int
	relevanceScore *float64
}

type bridgeWitnessEdge struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Type       string `json:"type"`
	Annotation string `json:"annotation"`
}

type bridgeRegionRepresentative struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type bridgeRegionSummary struct {
	Representative bridgeRegionRepresentative `json:"representative"`
	Size           int                        `json:"size"`
}

type bridgeWitnessRegions struct {
	Incoming   *bridgeRegionSummary `json:"incoming"`
	Outgoing   *bridgeRegionSummary `json:"outgoing"`
	SameRegion bool                 `json:"same_region"`
}

type bridgeWitness struct {
	Incoming bridgeWitnessEdge    `json:"incoming"`
	Outgoing bridgeWitnessEdge    `json:"outgoing"`
	Regions  bridgeWitnessRegions `json:"regions"`
}

const defaultBridgeWitnessCap = 3

type bridgeRecord struct {
	ID             string          `json:"id"`
	Title          string          `json:"title"`
	Score          int             `json:"score"`
	RelevanceScore *float64        `json:"relevance_score"`
	Witnesses      []bridgeWitness `json:"witnesses"`
}

func bridgeWitnessEdgeLess(left, right bridgeWitnessEdge) bool {
	if left.ID != right.ID {
		return left.ID < right.ID
	}
	if left.Type != right.Type {
		return left.Type < right.Type
	}
	return left.Annotation < right.Annotation
}

type bridgeRegionIdentity struct {
	region        *fullGraphRegion
	unclusteredID string
}

type bridgeRegionPair struct {
	incoming bridgeRegionIdentity
	outgoing bridgeRegionIdentity
}

type bridgeWitnessSelection struct {
	incomingSortKey string
	outgoingSortKey string
	witness         bridgeWitness
}

func buildBridgeRecords(candidates []bridgeCandidate, notes []*note.Note) []bridgeRecord {
	titles := make(map[string]string, len(notes))
	for _, n := range notes {
		titles[n.ID] = n.Title
	}

	incoming := make(map[string][]bridgeWitnessEdge, len(candidates))
	outgoing := make(map[string][]bridgeWitnessEdge, len(candidates))
	for _, n := range notes {
		for _, link := range n.Links {
			incoming[link.TargetID] = append(incoming[link.TargetID], bridgeWitnessEdge{
				ID: n.ID, Title: n.Title, Type: link.Type, Annotation: link.Annotation,
			})
			outgoing[n.ID] = append(outgoing[n.ID], bridgeWitnessEdge{
				ID: link.TargetID, Title: titles[link.TargetID], Type: link.Type, Annotation: link.Annotation,
			})
		}
	}
	for id := range incoming {
		sort.Slice(incoming[id], func(i, j int) bool { return bridgeWitnessEdgeLess(incoming[id][i], incoming[id][j]) })
	}
	for id := range outgoing {
		sort.Slice(outgoing[id], func(i, j int) bool { return bridgeWitnessEdgeLess(outgoing[id][i], outgoing[id][j]) })
	}

	regionsByNote := fullGraphRegionIndex(notes)
	regionIdentity := func(id string) (bridgeRegionIdentity, string) {
		if region := regionsByNote[id]; region != nil {
			return bridgeRegionIdentity{region: region}, region.representative.ID
		}
		// NUL cannot occur in a note ID, so this sentinel orders all missing
		// endpoints deterministically without exposing a durable region label.
		return bridgeRegionIdentity{unclusteredID: id}, "\x00unclustered:" + id
	}
	regionSummary := func(id string) *bridgeRegionSummary {
		region := regionsByNote[id]
		if region == nil {
			return nil
		}
		return &bridgeRegionSummary{
			Representative: bridgeRegionRepresentative{
				ID:    region.representative.ID,
				Title: region.representative.Title,
			},
			Size: region.size,
		}
	}

	records := make([]bridgeRecord, len(candidates))
	for i, candidate := range candidates {
		byRegionPair := make(map[bridgeRegionPair]bridgeWitnessSelection)
		for _, incomingEdge := range incoming[candidate.id] {
			incomingRegion, incomingSortKey := regionIdentity(incomingEdge.ID)
			for _, outgoingEdge := range outgoing[candidate.id] {
				outgoingRegion, outgoingSortKey := regionIdentity(outgoingEdge.ID)
				pair := bridgeRegionPair{incoming: incomingRegion, outgoing: outgoingRegion}
				selection := bridgeWitnessSelection{
					incomingSortKey: incomingSortKey,
					outgoingSortKey: outgoingSortKey,
					witness: bridgeWitness{
						Incoming: incomingEdge,
						Outgoing: outgoingEdge,
						Regions: bridgeWitnessRegions{
							Incoming:   regionSummary(incomingEdge.ID),
							Outgoing:   regionSummary(outgoingEdge.ID),
							SameRegion: incomingRegion.region != nil && incomingRegion.region == outgoingRegion.region,
						},
					},
				}
				current, exists := byRegionPair[pair]
				if !exists || bridgeWitnessPairLess(selection.witness, current.witness) {
					byRegionPair[pair] = selection
				}
			}
		}

		selections := make([]bridgeWitnessSelection, 0, len(byRegionPair))
		for _, selection := range byRegionPair {
			selections = append(selections, selection)
		}
		sort.Slice(selections, func(i, j int) bool {
			left, right := selections[i], selections[j]
			if left.incomingSortKey != right.incomingSortKey {
				return left.incomingSortKey < right.incomingSortKey
			}
			if left.outgoingSortKey != right.outgoingSortKey {
				return left.outgoingSortKey < right.outgoingSortKey
			}
			return bridgeWitnessPairLess(left.witness, right.witness)
		})
		if len(selections) > defaultBridgeWitnessCap {
			selections = selections[:defaultBridgeWitnessCap]
		}
		witnesses := make([]bridgeWitness, len(selections))
		for j, selection := range selections {
			witnesses[j] = selection.witness
		}
		records[i] = bridgeRecord{
			ID:             candidate.id,
			Title:          candidate.title,
			Score:          candidate.score,
			RelevanceScore: candidate.relevanceScore,
			Witnesses:      witnesses,
		}
	}
	return records
}

func bridgeWitnessPairLess(left, right bridgeWitness) bool {
	if bridgeWitnessEdgeLess(left.Incoming, right.Incoming) {
		return true
	}
	if bridgeWitnessEdgeLess(right.Incoming, left.Incoming) {
		return false
	}
	return bridgeWitnessEdgeLess(left.Outgoing, right.Outgoing)
}

func writeBridgeRecordsText(w io.Writer, records []bridgeRecord) {
	for i, record := range records {
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "%s  %s  (score %d)\n", record.ID, record.Title, record.Score)
		if record.RelevanceScore == nil {
			fmt.Fprintln(w, "  relevance: n/a")
		} else {
			fmt.Fprintf(w, "  relevance: %.6f\n", *record.RelevanceScore)
		}
		for j, witness := range record.Witnesses {
			fmt.Fprintf(w, "  crossing %d:\n", j+1)
			fmt.Fprintf(w, "    inbound edge: %s  %s -> %s  %s  (type: %q, annotation: %q)\n",
				witness.Incoming.ID, witness.Incoming.Title, record.ID, record.Title,
				witness.Incoming.Type, witness.Incoming.Annotation)
			fmt.Fprintf(w, "    outgoing edge: %s  %s -> %s  %s  (type: %q, annotation: %q)\n",
				record.ID, record.Title, witness.Outgoing.ID, witness.Outgoing.Title,
				witness.Outgoing.Type, witness.Outgoing.Annotation)
			writeBridgeRegionText(w, "incoming", witness.Regions.Incoming)
			writeBridgeRegionText(w, "outgoing", witness.Regions.Outgoing)
			fmt.Fprintf(w, "    same region: %t\n", witness.Regions.SameRegion)
		}
	}
}

func writeBridgeRegionText(w io.Writer, direction string, region *bridgeRegionSummary) {
	if region == nil {
		fmt.Fprintf(w, "    %s region: unclustered\n", direction)
		return
	}
	fmt.Fprintf(w, "    %s region: representative %s  %s; size: %d\n",
		direction, region.Representative.ID, region.Representative.Title, region.Size)
}
