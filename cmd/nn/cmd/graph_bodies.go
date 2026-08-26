package cmd

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"math"
	"sort"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
)

const (
	// graphBodiesPageMaxBytes leaves an explicit 2KB margin below a 50KB
	// transport boundary. The count includes compact JSON and its final newline.
	graphBodiesPageMaxBytes = 48_000
	// A valid UTF-8 byte can expand to at most six JSON bytes under encoding/json
	// (for example an ASCII control or an HTML-escaped '<', '>', or '&'). Keeping
	// source chunks at 7KB guarantees that one record plus its envelope fits;
	// exact encoded sizes are still measured when records are packed into pages.
	graphBodiesChunkMaxBytes = 7_000
)

type graphBodySegment struct {
	ID       string `json:"id"`
	Segment  int    `json:"segment"`
	Segments int    `json:"segments"`
	Body     string `json:"body"`
}

type graphBodiesPage struct {
	Snapshot string             `json:"snapshot"`
	Page     int                `json:"page"`
	Pages    int                `json:"pages"`
	NextPage int                `json:"next_page"`
	Segments []graphBodySegment `json:"segments"`
}

type graphBodiesSnapshotRequest struct {
	Version        int      `json:"version"`
	Focus          string   `json:"focus"`
	Depth          int      `json:"depth"`
	Direction      string   `json:"direction"`
	Links          []string `json:"links"`
	Statuses       []string `json:"statuses"`
	Representation string   `json:"representation"`
}

func newGraphBodiesCmd(state *rootState) *cobra.Command {
	var focus string
	var depth int
	var direction string
	var links string
	var statuses string
	var representation string
	var page int
	var snapshot string

	cmd := &cobra.Command{
		Use:   "bodies",
		Short: "Lossless paginated JSON bodies for a traversal-filtered graph set",
		Long: "Transport the stored Markdown bodies for the same traversal-filtered note set as graph show. " +
			"Output is compact, lossless JSON split into pages safely below 50KB. " +
			"Fetch page 1 first, then pass its snapshot to every later page.",
		Example: "  nn graph bodies --focus <id> --depth 1 --direction both --page 1\n" +
			"  nn graph bodies --focus <id> --depth 1 --direction both --page 2 --snapshot <sha256>",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if page < 1 {
				return fmt.Errorf("graph bodies: --page must be at least 1")
			}
			if page > 1 && snapshot == "" {
				return fmt.Errorf("graph bodies: --snapshot is required for page %d", page)
			}
			opts, err := newGraphTraversalOptions("graph bodies", direction, links, statuses, representation)
			if err != nil {
				return err
			}
			if focus == "" {
				for _, flag := range []string{"depth", "direction", "links", "status", "representation"} {
					if cmd.Flags().Changed(flag) {
						return fmt.Errorf("graph bodies: --%s requires --focus", flag)
					}
				}
			}

			notes, err := state.backend.List()
			if err != nil {
				return fmt.Errorf("graph bodies: %w", err)
			}
			selected, err := selectGraphBodyNotes(notes, focus, depth, opts)
			if err != nil {
				return err
			}
			for _, n := range selected {
				if !utf8.ValidString(n.Body) {
					return fmt.Errorf("graph bodies: note %q body is not valid UTF-8", n.ID)
				}
			}

			expectedSnapshot, err := graphBodiesSnapshot(notes, focus, depth, opts)
			if err != nil {
				return fmt.Errorf("graph bodies: snapshot: %w", err)
			}
			if snapshot != "" && snapshot != expectedSnapshot {
				return fmt.Errorf("graph bodies: stale or mismatched --snapshot; notebook or traversal changed")
			}

			segments := buildGraphBodySegments(selected)
			packed, err := packGraphBodySegments(expectedSnapshot, segments)
			if err != nil {
				return fmt.Errorf("graph bodies: %w", err)
			}
			if page > len(packed) {
				return fmt.Errorf("graph bodies: --page %d out of range (pages: %d)", page, len(packed))
			}
			response := graphBodiesPage{
				Snapshot: expectedSnapshot,
				Page:     page,
				Pages:    len(packed),
				Segments: packed[page-1],
			}
			if page < len(packed) {
				response.NextPage = page + 1
			}
			encoded, err := json.Marshal(response)
			if err != nil {
				return fmt.Errorf("graph bodies: encode page: %w", err)
			}
			if len(encoded)+1 > graphBodiesPageMaxBytes {
				return fmt.Errorf("graph bodies: encoded page is %d bytes (limit %d)", len(encoded)+1, graphBodiesPageMaxBytes)
			}
			if _, err := outWriter(cmd).Write(append(encoded, '\n')); err != nil {
				return fmt.Errorf("graph bodies: write page: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&focus, "focus", "", "Center note ID for ego-graph body selection")
	cmd.Flags().IntVar(&depth, "depth", 2, "BFS depth from focus note")
	cmd.Flags().StringVar(&direction, "direction", "outgoing", "Traversal direction: outgoing, incoming, or both")
	cmd.Flags().StringVar(&links, "links", "", "Comma-separated link types to traverse")
	cmd.Flags().StringVar(&statuses, "status", "", "Comma-separated note statuses to traverse")
	cmd.Flags().StringVar(&representation, "representation", "", "Representation required for traversed notes")
	cmd.Flags().IntVar(&page, "page", 1, "One-based page to return")
	cmd.Flags().StringVar(&snapshot, "snapshot", "", "Snapshot SHA-256 returned by page 1 (required for later pages)")
	return cmd
}

func selectGraphBodyNotes(notes []*note.Note, focus string, depth int, opts graphShowTraversalOptions) ([]*note.Note, error) {
	byID := make(map[string]*note.Note, len(notes))
	for _, n := range notes {
		byID[n.ID] = n
	}
	var selected []*note.Note
	if focus != "" {
		root := byID[focus]
		if root == nil {
			return nil, fmt.Errorf("graph bodies: note %q not found", focus)
		}
		entries := graphShowBFS(root, byID, depth, opts)
		selected = make([]*note.Note, 0, len(entries))
		for _, entry := range entries {
			selected = append(selected, entry.n)
		}
	} else {
		selected = append(selected, notes...)
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].ID < selected[j].ID })
	return selected, nil
}

func buildGraphBodySegments(notes []*note.Note) []graphBodySegment {
	var records []graphBodySegment
	for _, n := range notes {
		chunks := splitGraphBody(n.Body)
		for i, chunk := range chunks {
			records = append(records, graphBodySegment{
				ID:       n.ID,
				Segment:  i + 1,
				Segments: len(chunks),
				Body:     chunk,
			})
		}
	}
	return records
}

func splitGraphBody(body string) []string {
	if body == "" {
		return []string{""}
	}
	chunks := make([]string, 0, (len(body)+graphBodiesChunkMaxBytes-1)/graphBodiesChunkMaxBytes)
	for start := 0; start < len(body); {
		end := start + graphBodiesChunkMaxBytes
		if end >= len(body) {
			end = len(body)
		} else {
			for end > start && !utf8.RuneStart(body[end]) {
				end--
			}
		}
		chunks = append(chunks, body[start:end])
		start = end
	}
	return chunks
}

func packGraphBodySegments(snapshot string, records []graphBodySegment) ([][]graphBodySegment, error) {
	if len(records) == 0 {
		return [][]graphBodySegment{{}}, nil
	}
	pages := [][]graphBodySegment{{}}
	for _, record := range records {
		last := len(pages) - 1
		candidate := append(append([]graphBodySegment(nil), pages[last]...), record)
		fits, err := graphBodySegmentsFit(snapshot, candidate)
		if err != nil {
			return nil, err
		}
		if fits {
			pages[last] = candidate
			continue
		}
		fits, err = graphBodySegmentsFit(snapshot, []graphBodySegment{record})
		if err != nil {
			return nil, err
		}
		if !fits {
			return nil, fmt.Errorf("body segment for note %q cannot fit in one page", record.ID)
		}
		pages = append(pages, []graphBodySegment{record})
	}
	return pages, nil
}

func graphBodySegmentsFit(snapshot string, segments []graphBodySegment) (bool, error) {
	// Max-width integers make packing independent of the eventual number of
	// pages and leave every real envelope no larger than the measured candidate.
	candidate := graphBodiesPage{
		Snapshot: snapshot,
		Page:     math.MaxInt64,
		Pages:    math.MaxInt64,
		NextPage: math.MaxInt64,
		Segments: segments,
	}
	encoded, err := json.Marshal(candidate)
	if err != nil {
		return false, err
	}
	return len(encoded)+1 <= graphBodiesPageMaxBytes, nil
}

func graphBodiesSnapshot(notes []*note.Note, focus string, depth int, opts graphShowTraversalOptions) (string, error) {
	sortedNotes := append([]*note.Note(nil), notes...)
	sort.Slice(sortedNotes, func(i, j int) bool { return sortedNotes[i].ID < sortedNotes[j].ID })
	h := sha256.New()
	writeSnapshotPart(h, []byte("nn graph bodies snapshot v1\x00"))
	for _, n := range sortedNotes {
		data, err := n.Marshal()
		if err != nil {
			return "", fmt.Errorf("marshal note %q: %w", n.ID, err)
		}
		writeSnapshotPart(h, []byte(n.ID))
		writeSnapshotPart(h, data)
	}
	request := graphBodiesSnapshotRequest{
		Version:        1,
		Focus:          focus,
		Depth:          depth,
		Direction:      opts.direction,
		Links:          sortedGraphTraversalLinks(opts),
		Statuses:       sortedGraphTraversalStatuses(opts),
		Representation: opts.representation,
	}
	requestData, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	writeSnapshotPart(h, requestData)
	return hex.EncodeToString(h.Sum(nil)), nil
}

func writeSnapshotPart(h hash.Hash, data []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(data)))
	_, _ = h.Write(length[:])
	_, _ = h.Write(data)
}

func sortedGraphTraversalLinks(opts graphShowTraversalOptions) []string {
	values := make([]string, 0, len(opts.linkTypes))
	for value := range opts.linkTypes {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func sortedGraphTraversalStatuses(opts graphShowTraversalOptions) []string {
	values := make([]string, 0, len(opts.statuses))
	for value := range opts.statuses {
		values = append(values, string(value))
	}
	sort.Strings(values)
	return values
}
