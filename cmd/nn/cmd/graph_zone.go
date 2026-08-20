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
//   - TOP    (what ego answers to): governs (either dir), refines/extends/grounded-by OUT
//   - BOTTOM (what builds on ego):  refines/extends/grounded-by/supports IN
//   - LEFT   (tension):             contradicts, questions (either dir)
//   - RIGHT  (lateral):             source-of, requires (either dir)
func zoneOf(linkType string, dir linkDir) zone {
	switch linkType {
	case "contradicts", "questions":
		return zoneLeft
	case "source-of", "requires":
		return zoneRight
	case "governs":
		return zoneTop
	case "refines", "extends", "grounded-by":
		if dir == dirOut {
			return zoneTop
		}
		return zoneBottom
	case "supports":
		if dir == dirIn {
			return zoneBottom
		}
		return zoneNone
	default:
		return zoneNone
	}
}
