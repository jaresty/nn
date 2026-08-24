package cmd

import (
	"fmt"
	"io"

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

type bridgeRecord struct {
	ID             string        `json:"id"`
	Title          string        `json:"title"`
	Score          int           `json:"score"`
	RelevanceScore *float64      `json:"relevance_score"`
	Witness        bridgeWitness `json:"witness"`
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

func buildBridgeRecords(candidates []bridgeCandidate, notes []*note.Note) []bridgeRecord {
	titles := make(map[string]string, len(notes))
	for _, n := range notes {
		titles[n.ID] = n.Title
	}

	incoming := make(map[string]bridgeWitnessEdge, len(candidates))
	outgoing := make(map[string]bridgeWitnessEdge, len(candidates))
	for _, n := range notes {
		for _, link := range n.Links {
			incomingEdge := bridgeWitnessEdge{ID: n.ID, Title: n.Title, Type: link.Type, Annotation: link.Annotation}
			if current, ok := incoming[link.TargetID]; !ok || bridgeWitnessEdgeLess(incomingEdge, current) {
				incoming[link.TargetID] = incomingEdge
			}
			outgoingEdge := bridgeWitnessEdge{ID: link.TargetID, Title: titles[link.TargetID], Type: link.Type, Annotation: link.Annotation}
			if current, ok := outgoing[n.ID]; !ok || bridgeWitnessEdgeLess(outgoingEdge, current) {
				outgoing[n.ID] = outgoingEdge
			}
		}
	}

	regionsByNote := fullGraphRegionIndex(notes)
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
		incomingEdge := incoming[candidate.id]
		outgoingEdge := outgoing[candidate.id]
		incomingRegion := regionsByNote[incomingEdge.ID]
		outgoingRegion := regionsByNote[outgoingEdge.ID]
		records[i] = bridgeRecord{
			ID:             candidate.id,
			Title:          candidate.title,
			Score:          candidate.score,
			RelevanceScore: candidate.relevanceScore,
			Witness: bridgeWitness{
				Incoming: incomingEdge,
				Outgoing: outgoingEdge,
				Regions: bridgeWitnessRegions{
					Incoming:   regionSummary(incomingEdge.ID),
					Outgoing:   regionSummary(outgoingEdge.ID),
					SameRegion: incomingRegion != nil && incomingRegion == outgoingRegion,
				},
			},
		}
	}
	return records
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
		fmt.Fprintf(w, "  inbound edge: %s  %s -> %s  %s  (type: %q, annotation: %q)\n",
			record.Witness.Incoming.ID, record.Witness.Incoming.Title, record.ID, record.Title,
			record.Witness.Incoming.Type, record.Witness.Incoming.Annotation)
		fmt.Fprintf(w, "  outgoing edge: %s  %s -> %s  %s  (type: %q, annotation: %q)\n",
			record.ID, record.Title, record.Witness.Outgoing.ID, record.Witness.Outgoing.Title,
			record.Witness.Outgoing.Type, record.Witness.Outgoing.Annotation)
		writeBridgeRegionText(w, "incoming", record.Witness.Regions.Incoming)
		writeBridgeRegionText(w, "outgoing", record.Witness.Regions.Outgoing)
		fmt.Fprintf(w, "  same region: %t\n", record.Witness.Regions.SameRegion)
	}
}

func writeBridgeRegionText(w io.Writer, direction string, region *bridgeRegionSummary) {
	if region == nil {
		fmt.Fprintf(w, "  %s region: unclustered\n", direction)
		return
	}
	fmt.Fprintf(w, "  %s region: representative %s  %s; size: %d\n",
		direction, region.Representative.ID, region.Representative.Title, region.Size)
}
