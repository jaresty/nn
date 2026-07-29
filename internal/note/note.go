// Package note provides the core Note type, ID generation, and
// frontmatter parsing/serialisation for the nn Zettelkasten CLI.
package note

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	"gopkg.in/yaml.v3"
)

// ── Types ────────────────────────────────────────────────────────────────────

// Type classifies the intellectual content of a note.
type Type string

const (
	TypeConcept     Type = "concept"
	TypeArgument    Type = "argument"
	TypeModel       Type = "model"
	TypeHypothesis  Type = "hypothesis"
	TypeObservation Type = "observation"
	TypeQuestion    Type = "question"
	TypeProtocol    Type = "protocol"
)

// KnownLinkTypes is the canonical set of link relationship types.
// nn link --type warns when the type is not in this set.
var KnownLinkTypes = map[string]bool{
	"refines":     true,
	"contradicts": true,
	"source-of":   true,
	"extends":     true,
	"supports":    true,
	"questions":   true,
	"governs":     true,
	"requires":    true,
	"grounded-by": true,
}

// LinkTypeDescriptions gives a one-line semantic definition for each known link type.
// Use these to choose the correct type before calling nn link.
var LinkTypeDescriptions = map[string]string{
	"refines":     "The source sharpens or narrows the target's claim without replacing it — use when adding precision or a sub-case.",
	"contradicts": "The source directly opposes the target's claim — use when two notes cannot both be true.",
	"source-of":   "The target is derived from or authored by the source — use for evidence, citations, or origin relationships.",
	"extends":     "The source adds structure or scope to the target without replacing it — use when building on top of an existing model.",
	"supports":    "The source corroborates the target's claim — use for independent evidence that strengthens but is not constitutive of the target.",
	"questions":   "The source raises an unresolved challenge to the target — use when the target's claim is uncertain or contested.",
	"governs":     "The source is an operating protocol that constrains how the target (or its domain) is acted on — use only for protocol notes.",
	"requires":     "The source cannot be acted on until the target is complete — use for task dependency, not conceptual dependency.",
	"grounded-by": "The source claim depends on the target observation as its evidential basis — use when removing the target would make the source claim ungrounded (stronger than supports, which is corroborative only).",
}

// LinkTypeOrder is the canonical display order for link types.
var LinkTypeOrder = []string{
	"refines", "contradicts", "source-of", "extends", "supports", "grounded-by", "questions", "governs", "requires",
}

// LinkTypeWarnings gives a confirmation requirement for link types with semantic hazards.
// Empty string means no warning.
var LinkTypeWarnings = map[string]string{
	"governs":  "governs creates a binding operating constraint visible at session start — only use when you intend the source note to act as an active protocol that governs LLM behavior",
	"requires": "confirm both notes are action-bearing (have checkboxes or represent a work item) before linking",
}

// IsKnownLinkType reports whether t is in the canonical link type set.
func IsKnownLinkType(t string) bool {
	return t == "" || KnownLinkTypes[t]
}

// ValidTypes returns the list of recognised note type strings.
func ValidTypes() []string {
	return []string{"concept", "argument", "model", "hypothesis", "observation", "question", "protocol"}
}

// IsValid reports whether t is one of the recognised note types.
func (t Type) IsValid() bool {
	switch t {
	case TypeConcept, TypeArgument, TypeModel, TypeHypothesis, TypeObservation, TypeQuestion, TypeProtocol:
		return true
	}
	return false
}

// Status is the review status of a note.
type Status string

const (
	StatusDraft     Status = "draft"
	StatusReviewed  Status = "reviewed"
	StatusPermanent Status = "permanent"
)

// IsValid reports whether s is one of the recognised note statuses.
func (s Status) IsValid() bool {
	switch s {
	case StatusDraft, StatusReviewed, StatusPermanent:
		return true
	}
	return false
}

// Link is an annotated outgoing link from one note to another.
type Link struct {
	TargetID   string
	Annotation string
	Type       string // optional relationship type, e.g. "refines", "contradicts"
	Status     string // "draft" or "reviewed"; empty = reviewed (backward-compat for old links)
}

// Note is the in-memory representation of a single Zettelkasten note.
type Note struct {
	// Frontmatter fields
	ID       string
	Title    string
	Type     Type
	Status   Status
	Tags        []string
	AppliesWhen string
	ExpiresWhen string
	Expires     *time.Time
	Created     time.Time
	Modified    time.Time

	// Body is the Markdown content between the frontmatter and the ## Links section.
	Body string

	// Links are parsed from the ## Links section.
	Links []Link
}

// ── ID generation ─────────────────────────────────────────────────────────────

var (
	idMu    sync.Mutex
	usedIDs = make(map[string]bool)
)

// GenerateID returns a unique note ID in the format <14-digit-timestamp>-<4-digit-random>.
// crypto/rand provides the suffix. A process-local set detects and retries on the rare
// within-process collision (e.g. 200 concurrent goroutines in the same second).
func GenerateID() string {
	idMu.Lock()
	defer idMu.Unlock()
	for {
		ts := time.Now().UTC().Format("20060102150405")
		n, err := rand.Int(rand.Reader, big.NewInt(10000))
		if err != nil {
			panic(fmt.Sprintf("note.GenerateID: crypto/rand: %v", err))
		}
		id := fmt.Sprintf("%s-%04d", ts, n.Int64())
		if !usedIDs[id] {
			usedIDs[id] = true
			return id
		}
	}
}

// ── Filename ──────────────────────────────────────────────────────────────────

// Filename returns the canonical filename for this note: <id>-<slug>.md
func (n *Note) Filename() string {
	return n.ID + "-" + slugify(n.Title) + ".md"
}

// slugify converts a title to a URL-safe lowercase slug.
func slugify(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash {
			b.WriteRune('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// ── Parsing ───────────────────────────────────────────────────────────────────

// frontmatterYAML mirrors the YAML structure used in note files.
type frontmatterYAML struct {
	ID       string    `yaml:"id"`
	Title    string    `yaml:"title"`
	Type     string    `yaml:"type"`
	Status   string    `yaml:"status"`
	Tags        []string  `yaml:"tags"`
	AppliesWhen string     `yaml:"applies_when,omitempty"`
	ExpiresWhen string     `yaml:"expires_when,omitempty"`
	Expires     *time.Time `yaml:"expires,omitempty"`
	Created     time.Time  `yaml:"created"`
	Modified    time.Time  `yaml:"modified"`
}

var (
	// linkLineRE optionally captures [type] and {status} between ]] and —.
	// Groups: 1=targetID, 2=type, 3=status, 4=annotation
	linkLineRE = regexp.MustCompile(`^\s*-\s+\[\[([^\]]+)\]\](?:\s*\[([^\]]+)\])?(?:\s*\{([^}]+)\})?\s*—\s*(.+)$`)
	bareLinkRE = regexp.MustCompile(`^\s*-\s+\[\[([^\]]+)\]\]\s*$`)
)

// Parse reads a Markdown file (with YAML frontmatter) and returns a Note.
// Returns an error if the frontmatter is invalid, the type is missing, or a
// bare link (without annotation) is found in the ## Links section.
func Parse(data []byte) (*Note, error) {
	fm, body, err := splitFrontmatter(data)
	if err != nil {
		return nil, fmt.Errorf("note.Parse: frontmatter: %w", err)
	}

	var raw frontmatterYAML
	if err := yaml.Unmarshal(fm, &raw); err != nil {
		return nil, fmt.Errorf("note.Parse: yaml: %w", err)
	}

	if raw.Type == "" {
		return nil, fmt.Errorf("note.Parse: type field is required")
	}
	noteType := Type(raw.Type)
	if !noteType.IsValid() {
		return nil, fmt.Errorf("note.Parse: invalid type %q", raw.Type)
	}
	noteStatus := Status(raw.Status)
	if raw.Status != "" && !noteStatus.IsValid() {
		return nil, fmt.Errorf("note.Parse: invalid status %q", raw.Status)
	}
	if raw.Status == "" {
		noteStatus = StatusDraft
	}

	noteBody, links, err := parseBody(body)
	if err != nil {
		return nil, fmt.Errorf("note.Parse: links: %w", err)
	}

	return &Note{
		ID:       raw.ID,
		Title:    raw.Title,
		Type:     noteType,
		Status:   noteStatus,
		Tags:        raw.Tags,
		AppliesWhen: raw.AppliesWhen,
		ExpiresWhen: raw.ExpiresWhen,
		Expires:     raw.Expires,
		Created:     raw.Created,
		Modified:    raw.Modified,
		Body:     noteBody,
		Links:    links,
	}, nil
}

// splitFrontmatter separates the YAML frontmatter (between --- markers) from the body.
func splitFrontmatter(data []byte) (fm []byte, body []byte, err error) {
	const sep = "---"
	s := string(data)
	if !strings.HasPrefix(s, sep) {
		return nil, nil, fmt.Errorf("no frontmatter found")
	}
	// Find the closing ---
	rest := s[len(sep):]
	idx := strings.Index(rest, "\n"+sep)
	if idx == -1 {
		return nil, nil, fmt.Errorf("frontmatter not closed")
	}
	fmStr := rest[:idx]
	bodyStr := rest[idx+len("\n"+sep):]
	// Strip optional newline after closing ---
	bodyStr = strings.TrimPrefix(bodyStr, "\n")
	return []byte(fmStr), []byte(bodyStr), nil
}

// parseBody splits the body into content text and ## Links section.
func parseBody(data []byte) (body string, links []Link, err error) {
	const linkSection = "\n## Links\n"
	s := string(data)

	idx := strings.Index(s, linkSection)
	if idx == -1 {
		// No links section
		return strings.TrimRight(s, "\n"), nil, nil
	}

	bodyPart := strings.TrimRight(s[:idx], "\n")
	linksPart := s[idx+len(linkSection):]

	for _, line := range strings.Split(linksPart, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		if m := linkLineRE.FindStringSubmatch(line); m != nil {
			links = append(links, Link{
				TargetID:   m[1],
				Type:       m[2],
				Status:     m[3],
				Annotation: strings.TrimSpace(m[4]),
			})
			continue
		}
		if bareLinkRE.MatchString(line) {
			return "", nil, fmt.Errorf("bare link without annotation: %q", line)
		}
	}
	return bodyPart, links, nil
}

// ── Serialisation ─────────────────────────────────────────────────────────────

// Marshal serialises the note back to Markdown with YAML frontmatter.
func (n *Note) Marshal() ([]byte, error) {
	raw := frontmatterYAML{
		ID:          n.ID,
		Title:       n.Title,
		Type:        string(n.Type),
		Status:      string(n.Status),
		Tags:        n.Tags,
		AppliesWhen: n.AppliesWhen,
		ExpiresWhen: n.ExpiresWhen,
		Expires:     n.Expires,
		Created:     n.Created,
		Modified:    n.Modified,
	}

	fmBytes, err := yaml.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("note.Marshal: yaml: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(fmBytes)
	buf.WriteString("---\n")
	if n.Body != "" {
		buf.WriteString("\n")
		buf.WriteString(n.Body)
		buf.WriteString("\n")
	}
	if len(n.Links) > 0 {
		buf.WriteString("\n## Links\n\n")
		for _, lnk := range n.Links {
			switch {
			case lnk.Type != "" && lnk.Status != "":
				fmt.Fprintf(&buf, "- [[%s]] [%s] {%s} — %s\n", lnk.TargetID, lnk.Type, lnk.Status, lnk.Annotation)
			case lnk.Type != "":
				fmt.Fprintf(&buf, "- [[%s]] [%s] — %s\n", lnk.TargetID, lnk.Type, lnk.Annotation)
			case lnk.Status != "":
				fmt.Fprintf(&buf, "- [[%s]] {%s} — %s\n", lnk.TargetID, lnk.Status, lnk.Annotation)
			default:
				fmt.Fprintf(&buf, "- [[%s]] — %s\n", lnk.TargetID, lnk.Annotation)
			}
		}
	}
	return buf.Bytes(), nil
}
