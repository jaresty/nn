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
	Heading      string
	HeadingLevel int
	SectionStart int
	LineStart    int
	LineEnd      int
	Text         string
}

type readContextOverviewSection struct {
	Heading    string `json:"heading"`
	Level      int    `json:"level"`
	LineStart  int    `json:"line_start"`
	LineEnd    int    `json:"line_end"`
	ChunkStart int    `json:"chunk_start"`
	ChunkEnd   int    `json:"chunk_end"`
	Current    bool   `json:"current"`
	Evidence   string `json:"evidence"`
}

type readContextOverviewResponse struct {
	Mode         string                       `json:"mode"`
	Snapshot     string                       `json:"snapshot"`
	Source       string                       `json:"source"`
	CurrentChunk int                          `json:"current_chunk"`
	Chunks       int                          `json:"chunks"`
	Page         int                          `json:"page"`
	Pages        int                          `json:"pages"`
	PreviousPage int                          `json:"previous_page"`
	NextPage     int                          `json:"next_page"`
	Coverage     string                       `json:"coverage"`
	Sections     []readContextOverviewSection `json:"sections"`
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
	var overview bool
	var page int

	cmd := &cobra.Command{
		Use:   "read-context <file>",
		Short: "Read one bounded document passage with generated notebook relevance",
		Long: "Return one heading-aware passage from a UTF-8 Markdown or text file plus bounded BM25-ranked notebook candidates, " +
			"or use --overview for a paginated source-order document structure map. Generated relevance is not a stored relationship and the candidate set is not exhaustive. " +
			"Fetch passage chunk 1 or overview page 1 first, then pass its snapshot to later chunks or pages.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if chunk < 1 {
				return fmt.Errorf("read-context: --chunk must be at least 1")
			}
			if page < 1 {
				return fmt.Errorf("read-context: --page must be at least 1")
			}
			if chunkBytes < readContextMinChunkBytes || chunkBytes > readContextMaxChunkBytes {
				return fmt.Errorf("read-context: --chunk-bytes must be between %d and %d", readContextMinChunkBytes, readContextMaxChunkBytes)
			}
			if matchLimit < 0 || matchLimit > readContextMaxMatches {
				return fmt.Errorf("read-context: --matches must be between 0 and %d", readContextMaxMatches)
			}
			if !overview && cmd.Flags().Changed("page") {
				return fmt.Errorf("read-context: --page requires --overview")
			}
			if !overview && chunk > 1 && snapshot == "" {
				return fmt.Errorf("read-context: --snapshot is required for chunk %d", chunk)
			}
			if overview && page > 1 && snapshot == "" {
				return fmt.Errorf("read-context: --snapshot is required for overview page %d", page)
			}
			if snapshot != "" && !overview && !cmd.Flags().Changed("chunk") {
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
			if overview {
				sections := buildReadContextOverview(passages, chunk)
				pages, err := paginateReadContextOverview(path, expectedSnapshot, chunk, len(passages), sections)
				if err != nil {
					return err
				}
				if page > len(pages) {
					return fmt.Errorf("read-context: --page %d out of range (pages: %d)", page, len(pages))
				}
				response := readContextOverviewResponse{
					Mode: "overview", Snapshot: expectedSnapshot, Source: path,
					CurrentChunk: chunk, Chunks: len(passages), Page: page, Pages: len(pages),
					Coverage: "document-structure only; excludes generated relevance and stored graph relationships",
					Sections: pages[page-1],
				}
				if page > 1 {
					response.PreviousPage = page - 1
				}
				if page < len(pages) {
					response.NextPage = page + 1
				}
				return writeReadContextJSON(cmd, response)
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
			return writeReadContextJSON(cmd, response)
		},
	}
	cmd.Flags().IntVar(&chunk, "chunk", 1, "One-based document passage to return")
	cmd.Flags().StringVar(&snapshot, "snapshot", "", "Source snapshot returned by chunk 1 or overview page 1")
	cmd.Flags().IntVar(&chunkBytes, "chunk-bytes", readContextDefaultChunkBytes, "Target source bytes per passage (256-6000)")
	cmd.Flags().IntVar(&matchLimit, "matches", 5, "Maximum generated-relevance notebook candidates (0-20)")
	cmd.Flags().BoolVar(&overview, "overview", false, "Return a paginated source-order document structure overview")
	cmd.Flags().IntVar(&page, "page", 1, "One-based overview page to return (requires --overview)")
	return cmd
}

func writeReadContextJSON(cmd *cobra.Command, response any) error {
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
}

func buildReadContextOverview(passages []readContextPassage, currentChunk int) []readContextOverviewSection {
	sections := make([]readContextOverviewSection, 0, len(passages))
	for i, passage := range passages {
		chunk := i + 1
		heading := passage.Heading
		if heading == "" {
			heading = "Document"
		}
		if len(sections) > 0 && sections[len(sections)-1].LineStart == passage.SectionStart {
			section := &sections[len(sections)-1]
			section.LineEnd = passage.LineEnd
			section.ChunkEnd = chunk
			section.Current = section.Current || chunk == currentChunk
			continue
		}
		sections = append(sections, readContextOverviewSection{
			Heading: heading, Level: passage.HeadingLevel,
			LineStart: passage.LineStart, LineEnd: passage.LineEnd,
			ChunkStart: chunk, ChunkEnd: chunk, Current: chunk == currentChunk,
			Evidence: "document-structure",
		})
	}
	return sections
}

func paginateReadContextOverview(source, snapshot string, currentChunk, chunks int, sections []readContextOverviewSection) ([][]readContextOverviewSection, error) {
	pages := [][]readContextOverviewSection{{}}
	for _, section := range sections {
		pageIndex := len(pages) - 1
		candidate := append(append([]readContextOverviewSection{}, pages[pageIndex]...), section)
		probe := readContextOverviewResponse{
			Mode: "overview", Snapshot: snapshot, Source: source,
			CurrentChunk: currentChunk, Chunks: chunks,
			Page: pageIndex + 1, Pages: len(sections), PreviousPage: pageIndex, NextPage: pageIndex + 2,
			Coverage: "document-structure only; excludes generated relevance and stored graph relationships",
			Sections: candidate,
		}
		encoded, err := json.Marshal(probe)
		if err != nil {
			return nil, fmt.Errorf("read-context: encode overview: %w", err)
		}
		if len(encoded)+1 <= readContextPageMaxBytes {
			pages[pageIndex] = candidate
			continue
		}
		if len(pages[pageIndex]) == 0 {
			return nil, fmt.Errorf("read-context: one overview section exceeds encoded response limit %d", readContextPageMaxBytes)
		}
		pages = append(pages, []readContextOverviewSection{section})
	}
	return pages, nil
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
		level   int
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
			if heading, level := readContextATXHeading(trimmed); heading != "" {
				if i > sections[len(sections)-1].start {
					sections[len(sections)-1].end = i
					sections = append(sections, section{heading: heading, level: level, start: i})
				} else {
					sections[len(sections)-1].heading = heading
					sections[len(sections)-1].level = level
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
			passages = append(passages, readContextPassage{Heading: sec.heading, HeadingLevel: sec.level, SectionStart: sec.start + 1, LineStart: start + 1, LineEnd: end, Text: b.String()})
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

func readContextATXHeading(line string) (string, int) {
	if !strings.HasPrefix(line, "#") {
		return "", 0
	}
	i := 0
	for i < len(line) && line[i] == '#' {
		i++
	}
	if i == 0 || i > 6 || i >= len(line) || line[i] != ' ' {
		return "", 0
	}
	return strings.TrimSpace(strings.TrimRight(strings.TrimSpace(line[i+1:]), "#")), i
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
