package cmd

// zone is the directional screen region a related node occupies in the graph
// viewer's "Zoned" ego layout. Zones are assigned from a node's link
// relationship to the focused ego node.
type zone string

const (
	zoneTop    zone = "top"    // what the ego depends on / answers to
	zoneBottom zone = "bottom" // what builds on the ego
	zoneLeft   zone = "left"   // tension: disagreement / open questions
	zoneRight  zone = "right"  // lateral: evidence provenance / task edges
	zoneNone   zone = ""       // no directional placement
)

// linkDir is the direction of a link relative to the focused ego node.
//   - dirOut: the ego links to the other node (ego -> N)
//   - dirIn:  the other node links to the ego (N -> ego)
type linkDir string

const (
	dirOut linkDir = "out"
	dirIn  linkDir = "in"
)

// zoneOf maps a (linkType, direction-relative-to-ego) pair to the screen zone
// the related node should occupy in the Zoned ego layout.
//
// Mapping (see note 20260820154156-6576):
//   - TOP    (what ego answers to): governs/supports IN, refines/extends/grounded-by OUT
//   - BOTTOM (what builds on ego):  governs/supports OUT, refines/extends/grounded-by IN
//   - LEFT   (tension):             contradicts, questions (either dir)
//   - RIGHT  (lateral):             source-of, requires, follows (either dir)
func zoneOf(linkType string, dir linkDir) zone {
	switch linkType {
	case "contradicts", "questions":
		return zoneLeft
	case "source-of", "requires", "follows", "related":
		// source-of/requires/follows are lateral; "related" is a legacy/generic
		// association with no clear direction, so treat it as lateral too.
		return zoneRight
	case "governs":
		// governs points from authority to governed. If a note governs the ego
		// (dirIn), the ego answers to it -> TOP. If the ego governs the note
		// (dirOut), that note is subordinate to the ego -> BOTTOM.
		if dir == dirIn {
			return zoneTop
		}
		return zoneBottom
	case "refines", "extends", "grounded-by":
		if dir == dirOut {
			return zoneTop
		}
		return zoneBottom
	case "supports":
		// supports points from corroborating evidence to the supported claim.
		// The claim answers to incoming evidence (TOP); the supported claim
		// builds on outgoing evidence (BOTTOM).
		if dir == dirIn {
			return zoneTop
		}
		return zoneBottom
	default:
		return zoneNone
	}
}
