package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"
)

const (
	readContextPageMaxBytes      = 48_000
	readContextDefaultChunkBytes = 4_000
	readContextMinChunkBytes     = 256
	readContextMaxChunkBytes     = 6_000
	readContextMaxMatches        = 20
)

type readContextPassage struct {
	Heading   string
	LineStart int
	LineEnd   int
	Text      string
}

type readContextMatch struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	Type         string  `json:"type"`
	Status       string  `json:"status"`
	Score        float64 `json:"score"`
	MatchReason  string  `json:"match_reason"`
	Excerpt      string  `json:"excerpt"`
	Relationship string  `json:"relationship"`
}

type readContextResponse struct {
	Snapshot      string             `json:"snapshot"`
	Source        string             `json:"source"`
	Chunk         int                `json:"chunk"`
	Chunks        int                `json:"chunks"`
	PreviousChunk int                `json:"previous_chunk"`
	NextChunk     int                `json:"next_chunk"`
	Heading       string             `json:"heading,omitempty"`
	LineStart     int                `json:"line_start"`
	LineEnd       int                `json:"line_end"`
	Passage       string             `json:"passage"`
	Coverage      string             `json:"coverage"`
	Matches       []readContextMatch `json:"matches"`
}

func newReadContextCmd(state *rootState) *cobra.Command {
	var chunk int
	var snapshot string
	var chunkBytes int
	var matchLimit int

	cmd := &cobra.Command{
		Use:   "read-context <file>",
		Short: "Read one bounded document passage with generated notebook relevance",
		Long: "Return one heading-aware passage from a UTF-8 Markdown or text file plus bounded BM25-ranked notebook candidates. " +
			"Generated relevance is not a stored relationship and the candidate set is not exhaustive. Fetch chunk 1 first, then pass its snapshot to later chunks.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if chunk < 1 {
				return fmt.Errorf("read-context: --chunk must be at least 1")
			}
			if chunkBytes < readContextMinChunkBytes || chunkBytes > readContextMaxChunkBytes {
				return fmt.Errorf("read-context: --chunk-bytes must be between %d and %d", readContextMinChunkBytes, readContextMaxChunkBytes)
			}
			if matchLimit < 0 || matchLimit > readContextMaxMatches {
				return fmt.Errorf("read-context: --matches must be between 0 and %d", readContextMaxMatches)
			}
			if chunk > 1 && snapshot == "" {
				return fmt.Errorf("read-context: --snapshot is required for chunk %d", chunk)
			}
			if snapshot != "" && !cmd.Flags().Changed("chunk") {
				return fmt.Errorf("read-context: --snapshot requires --chunk")
			}

			path := filepath.Clean(args[0])
			ext := strings.ToLower(filepath.Ext(path))
			switch ext {
			case ".md", ".markdown", ".txt":
			default:
				return fmt.Errorf("read-context: unsupported file extension %q (want .md, .markdown, or .txt)", ext)
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("read-context: read %s: %w", path, err)
			}
			if !utf8.Valid(data) {
				return fmt.Errorf("read-context: source is not valid UTF-8")
			}

			expectedSnapshot := readContextSnapshot(data, ext, chunkBytes)
			if snapshot != "" && snapshot != expectedSnapshot {
				return fmt.Errorf("read-context: stale or mismatched --snapshot; source or chunking options changed")
			}
			passages := splitReadContextPassages(string(data), ext != ".txt", chunkBytes)
			if chunk > len(passages) {
				return fmt.Errorf("read-context: --chunk %d out of range (chunks: %d)", chunk, len(passages))
			}
			selected := passages[chunk-1]
			matches, err := rankReadContextMatches(state, selected.Text, matchLimit)
			if err != nil {
				return fmt.Errorf("read-context: rank notebook context: %w", err)
			}
			response := readContextResponse{
				Snapshot:  expectedSnapshot,
				Source:    path,
				Chunk:     chunk,
				Chunks:    len(passages),
				Heading:   selected.Heading,
				LineStart: selected.LineStart,
				LineEnd:   selected.LineEnd,
				Passage:   selected.Text,
				Coverage:  "bounded generated relevance; not exhaustive and not a stored notebook relationship",
				Matches:   matches,
			}
			if chunk > 1 {
				response.PreviousChunk = chunk - 1
			}
			if chunk < len(passages) {
				response.NextChunk = chunk + 1
			}
			encoded, err := json.Marshal(response)
			if err != nil {
				return fmt.Errorf("read-context: encode response: %w", err)
			}
			if len(encoded)+1 > readContextPageMaxBytes {
				return fmt.Errorf("read-context: encoded response is %d bytes (limit %d)", len(encoded)+1, readContextPageMaxBytes)
			}
			if _, err := outWriter(cmd).Write(append(encoded, '\n')); err != nil {
				return fmt.Errorf("read-context: write response: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&chunk, "chunk", 1, "One-based document passage to return")
	cmd.Flags().StringVar(&snapshot, "snapshot", "", "Source snapshot returned by chunk 1 (required for later chunks)")
	cmd.Flags().IntVar(&chunkBytes, "chunk-bytes", readContextDefaultChunkBytes, "Target source bytes per passage (256-6000)")
	cmd.Flags().IntVar(&matchLimit, "matches", 5, "Maximum generated-relevance notebook candidates (0-20)")
	return cmd
}

func readContextSnapshot(data []byte, ext string, chunkBytes int) string {
	h := sha256.New()
	fmt.Fprintf(h, "nn.read-context/v1\x00%s\x00%d\x00", ext, chunkBytes)
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func splitReadContextPassages(text string, markdown bool, maxBytes int) []readContextPassage {
	if text == "" {
		return []readContextPassage{{LineStart: 1, LineEnd: 1, Text: ""}}
	}
	lines := strings.SplitAfter(text, "\n")
	type section struct {
		heading string
		start   int
		end     int
	}
	sections := []section{{start: 0}}
	if markdown {
		inFence := false
		for i, line := range lines {
			trimmed := strings.TrimSpace(strings.TrimSuffix(line, "\n"))
			if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
				inFence = !inFence
				continue
			}
			if inFence {
				continue
			}
			if heading := readContextATXHeading(trimmed); heading != "" {
				if i > sections[len(sections)-1].start {
					sections[len(sections)-1].end = i
					sections = append(sections, section{heading: heading, start: i})
				} else {
					sections[len(sections)-1].heading = heading
				}
			}
		}
	}
	sections[len(sections)-1].end = len(lines)
	var passages []readContextPassage
	for _, sec := range sections {
		if sec.end <= sec.start {
			continue
		}
		start := sec.start
		var b strings.Builder
		flush := func(end int) {
			if b.Len() == 0 {
				return
			}
			passages = append(passages, readContextPassage{Heading: sec.heading, LineStart: start + 1, LineEnd: end, Text: b.String()})
			b.Reset()
			start = end
		}
		for i := sec.start; i < sec.end; i++ {
			line := lines[i]
			if b.Len() > 0 && b.Len()+len(line) > maxBytes {
				flush(i)
			}
			for len(line) > maxBytes {
				cut := utf8SafePrefix(line, maxBytes)
				b.WriteString(cut)
				line = line[len(cut):]
				flush(i + 1)
			}
			b.WriteString(line)
		}
		flush(sec.end)
	}
	if len(passages) == 0 {
		return []readContextPassage{{LineStart: 1, LineEnd: 1, Text: ""}}
	}
	return passages
}

func readContextATXHeading(line string) string {
	if !strings.HasPrefix(line, "#") {
		return ""
	}
	i := 0
	for i < len(line) && line[i] == '#' {
		i++
	}
	if i == 0 || i > 6 || i >= len(line) || line[i] != ' ' {
		return ""
	}
	return strings.TrimSpace(strings.TrimRight(strings.TrimSpace(line[i+1:]), "#"))
}

func utf8SafePrefix(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(value[cut]) {
		cut--
	}
	if cut == 0 {
		_, size := utf8.DecodeRuneInString(value)
		cut = size
	}
	return value[:cut]
}

func truncateUTF8(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	return utf8SafePrefix(value, maxBytes-3) + "..."
}

func rankReadContextMatches(state *rootState, query string, limit int) ([]readContextMatch, error) {
	matches := []readContextMatch{}
	if limit == 0 || state.backend == nil || strings.TrimSpace(query) == "" {
		return matches, nil
	}
	notes, err := state.backend.List()
	if err != nil {
		return nil, err
	}
	scores := RankedByQuery(notes, notes, query, state.notebookDir)
	sort.SliceStable(notes, func(i, j int) bool {
		if scores[notes[i].ID] == scores[notes[j].ID] {
			return notes[i].ID < notes[j].ID
		}
		return scores[notes[i].ID] > scores[notes[j].ID]
	})
	maxScore := 0.0
	for _, n := range notes {
		if scores[n.ID] > maxScore {
			maxScore = scores[n.ID]
		}
	}
	for _, n := range notes {
		if len(matches) >= limit || scores[n.ID] <= 0 {
			break
		}
		score := scores[n.ID]
		if maxScore > 0 {
			score /= maxScore
		}
		matches = append(matches, readContextMatch{
			ID:           n.ID,
			Title:        truncateUTF8(n.Title, 240),
			Type:         string(n.Type),
			Status:       string(n.Status),
			Score:        score,
			MatchReason:  computeMatchReason(n, query),
			Excerpt:      truncateUTF8(extractExcerpt(n.Body, query), 320),
			Relationship: "generated-relevance",
		})
	}
	return matches, nil
}
