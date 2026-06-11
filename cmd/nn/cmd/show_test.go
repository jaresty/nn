package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: TestShowDailyCreationEmbedsYesterday — when resolveDailyNote creates today's note, it embeds yesterday's body as ### Yesterday section.
func TestShowDailyCreationEmbedsYesterday(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	yesterdayTitle := "Daily: " + yesterday
	yNote := newTestNoteForCLI(note.GenerateID(), yesterdayTitle, note.TypeObservation)
	yNote.Tags = []string{"daily"}
	yNote.Body = "## Done\n- fixed the thing"
	writeNoteFile(t, nbDir, yNote)

	out, err := execute("show", "daily")
	if err != nil {
		t.Fatalf("nn show daily: %v", err)
	}
	if !strings.Contains(out, "### Yesterday") {
		t.Errorf("nn show daily: want '### Yesterday' section in new note, got:\n%s", out)
	}
	if !strings.Contains(out, "fixed the thing") {
		t.Errorf("nn show daily: want yesterday body embedded, got:\n%s", out)
	}
}

// Assertion: TestShowGlobalCreatesTodayWhenAbsent — nn show --global creates and shows today's daily note when none exists.
func TestShowGlobalCreatesTodayWhenAbsent(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	yesterdayTitle := "Daily: " + yesterday
	yNote := newTestNoteForCLI(note.GenerateID(), yesterdayTitle, note.TypeObservation)
	yNote.Tags = []string{"daily"}
	yNote.Status = note.StatusPermanent
	yNote.Body = "## Done\n- session work"
	writeNoteFile(t, nbDir, yNote)

	today := time.Now().Format("2006-01-02")
	todayTitle := "Daily: " + today

	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, todayTitle) {
		t.Errorf("nn show --global: want today note %q created and shown, got:\n%s", todayTitle, out)
	}
}

// Assertion: TestShowDailyResolvesToToday — nn show daily resolves to today's Daily: YYYY-MM-DD note, not a stale one.
func TestShowDailyResolvesToToday(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	today := time.Now().Format("2006-01-02")
	todayTitle := "Daily: " + today
	n := newTestNoteForCLI(note.GenerateID(), todayTitle, note.TypeObservation)
	n.Tags = []string{"daily"}
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", "daily")
	if err != nil {
		t.Fatalf("nn show daily: %v", err)
	}
	if !strings.Contains(out, todayTitle) {
		t.Errorf("nn show daily: want output containing %q, got:\n%s", todayTitle, out)
	}
}

// Assertion: TestShowDailyCreatesNoteWhenAbsent — nn show daily creates a new Daily: YYYY-MM-DD note if none exists today.
func TestShowDailyCreatesNoteWhenAbsent(t *testing.T) {
	_, execute := setupNotebook(t)
	today := time.Now().Format("2006-01-02")
	todayTitle := "Daily: " + today

	out, err := execute("show", "daily")
	if err != nil {
		t.Fatalf("nn show daily (absent): %v", err)
	}
	if !strings.Contains(out, todayTitle) {
		t.Errorf("nn show daily: want created note with title %q, got:\n%s", todayTitle, out)
	}
}

// Assertion: TestDailyNoteTitleUsesLocalTime — daily note titles use local time, not UTC.
// Simulates a moment that is one date in UTC but the prior date locally (UTC-7 / PDT).
func TestDailyNoteTitleUsesLocalTime(t *testing.T) {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skip("timezone data unavailable")
	}
	// 2026-06-11T00:30:00Z = 2026-06-10 17:30 PDT — UTC date is 11th, local date is 10th.
	fixedUTC := time.Date(2026, 6, 11, 0, 30, 0, 0, time.UTC)
	fixedLocal := fixedUTC.In(loc)
	orig := nowFn
	nowFn = func() time.Time { return fixedLocal }
	t.Cleanup(func() { nowFn = orig })

	_, execute := setupNotebook(t)
	out, err := execute("show", "daily")
	if err != nil {
		t.Fatalf("nn show daily: %v", err)
	}
	localTitle := "Daily: " + fixedLocal.Format("2006-01-02") // "Daily: 2026-06-10"
	utcTitle := "Daily: " + fixedUTC.Format("2006-01-02")     // "Daily: 2026-06-11"
	if strings.Contains(out, utcTitle) {
		t.Errorf("daily note title used UTC date %q; want local date %q", utcTitle, localTitle)
	}
	if !strings.Contains(out, localTitle) {
		t.Errorf("daily note title missing local date %q, got:\n%s", localTitle, out)
	}
}

// Assertion: TestDailyYesterdayEmbedUsesLocalTime — yesterday's body is embedded using local date, not UTC.
func TestDailyYesterdayEmbedUsesLocalTime(t *testing.T) {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Skip("timezone data unavailable")
	}
	// fixedLocal = 2026-06-10 17:30 PDT. Local yesterday = 2026-06-09. UTC yesterday = 2026-06-10.
	fixedUTC := time.Date(2026, 6, 11, 0, 30, 0, 0, time.UTC)
	fixedLocal := fixedUTC.In(loc)
	orig := nowFn
	nowFn = func() time.Time { return fixedLocal }
	t.Cleanup(func() { nowFn = orig })

	nbDir, execute := setupNotebook(t)
	// Write a note titled "Daily: 2026-06-09" (local yesterday).
	localYesterdayTitle := "Daily: " + fixedLocal.AddDate(0, 0, -1).Format("2006-01-02")
	yNote := newTestNoteForCLI(note.GenerateID(), localYesterdayTitle, note.TypeObservation)
	yNote.Tags = []string{"daily"}
	yNote.Body = "## Done\n- local yesterday content"
	writeNoteFile(t, nbDir, yNote)

	out, err := execute("show", "daily")
	if err != nil {
		t.Fatalf("nn show daily: %v", err)
	}
	if !strings.Contains(out, "local yesterday content") {
		t.Errorf("daily note did not embed local yesterday body; got:\n%s", out)
	}
}

// Assertion: TestShowDailyCreatedNoteHasDailyTag — note created by nn show daily has tag daily.
func TestShowDailyCreatedNoteHasDailyTag(t *testing.T) {
	_, execute := setupNotebook(t)

	out, err := execute("show", "daily")
	if err != nil {
		t.Fatalf("nn show daily: %v", err)
	}
	if !strings.Contains(out, "daily") {
		t.Errorf("nn show daily: want 'daily' tag in output, got:\n%s", out)
	}
}

// Assertion: TestShowDailyCreatedNoteHasSevenDayExpiry — note created by nn show daily expires in 7 days.
func TestShowDailyCreatedNoteHasSevenDayExpiry(t *testing.T) {
	_, execute := setupNotebook(t)
	sevenDaysFromNow := time.Now().UTC().AddDate(0, 0, 7).Format("2006-01-02")

	out, err := execute("show", "daily")
	if err != nil {
		t.Fatalf("nn show daily: %v", err)
	}
	if !strings.Contains(out, sevenDaysFromNow) {
		t.Errorf("nn show daily: want expiry %q in output, got:\n%s", sevenDaysFromNow, out)
	}
}

// Assertion: TestShowFreshnessFresh — note modified 1 day ago shows freshness: fresh with age and hint
func TestShowFreshnessFresh(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Fresh Note", note.TypeConcept)
	n.Modified = time.Now().UTC().Add(-24 * time.Hour)
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", n.ID)
	if err != nil {
		t.Fatalf("nn show: %v", err)
	}
	if !strings.Contains(out, "freshness: fresh") {
		t.Errorf("want freshness: fresh in output, got:\n%s", out)
	}
	if !strings.Contains(out, "likely current") {
		t.Errorf("want 'likely current' hint in output, got:\n%s", out)
	}
	if !strings.Contains(out, "ago") {
		t.Errorf("want age 'ago' in output, got:\n%s", out)
	}
}

// Assertion: TestShowFreshnessAging — note modified 7 days ago shows freshness: aging with age and hint
func TestShowFreshnessAging(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Aging Note", note.TypeConcept)
	n.Modified = time.Now().UTC().Add(-7 * 24 * time.Hour)
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", n.ID)
	if err != nil {
		t.Fatalf("nn show: %v", err)
	}
	if !strings.Contains(out, "freshness: aging") {
		t.Errorf("want freshness: aging in output, got:\n%s", out)
	}
	if !strings.Contains(out, "may need recheck") {
		t.Errorf("want 'may need recheck' hint in output, got:\n%s", out)
	}
	if !strings.Contains(out, "ago") {
		t.Errorf("want age 'ago' in output, got:\n%s", out)
	}
}

// Assertion: TestShowFreshnessStale — note modified 30 days ago shows freshness: stale with age and hint
func TestShowFreshnessStale(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Stale Note", note.TypeConcept)
	n.Modified = time.Now().UTC().Add(-30 * 24 * time.Hour)
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", n.ID)
	if err != nil {
		t.Fatalf("nn show: %v", err)
	}
	if !strings.Contains(out, "freshness: stale") {
		t.Errorf("want freshness: stale in output, got:\n%s", out)
	}
	if !strings.Contains(out, "content may be outdated") {
		t.Errorf("want 'content may be outdated' hint in output, got:\n%s", out)
	}
	if !strings.Contains(out, "ago") {
		t.Errorf("want age 'ago' in output, got:\n%s", out)
	}
}

func TestShowNote(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Show Me", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", n.ID)
	if err != nil {
		t.Fatalf("nn show: %v", err)
	}
	if !strings.Contains(out, "Show Me") {
		t.Errorf("output %q does not contain title 'Show Me'", out)
	}
}

func TestShowNoteNotFound(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("show", "99999999999999-0000")
	if err == nil {
		t.Fatal("nn show nonexistent: want error, got nil")
	}
}

// Assertion: TestShowProtocolNoDerivationBlock — plain nn show on a protocol note does NOT include ## Protocols block.
// The derivation block is only appended once in nn show --global output.
func TestShowProtocolNoDerivationBlock(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "My Protocol", note.TypeProtocol)
	n.Body = "Do the thing before acting."
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", n.ID)
	if err != nil {
		t.Fatalf("nn show: %v", err)
	}
	if strings.Contains(out, "## Protocols") {
		t.Errorf("expected no '## Protocols' derivation block in individual protocol note output; got:\n%s", out)
	}
}

// Assertion: TestShowNonProtocolNoDerivation — nn show on a concept note does NOT include ## Protocols block.
func TestShowNonProtocolNoDerivation(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	proto := newTestNoteForCLI(note.GenerateID(), "My Protocol", note.TypeProtocol)
	proto.Body = "Do the thing."
	concept := newTestNoteForCLI(note.GenerateID(), "My Concept", note.TypeConcept)
	concept.Body = "A concept about things."
	writeNoteFile(t, nbDir, proto)
	writeNoteFile(t, nbDir, concept)

	out, err := execute("show", concept.ID)
	if err != nil {
		t.Fatalf("nn show: %v", err)
	}
	if strings.Contains(out, "## Protocols") {
		t.Errorf("expected no '## Protocols' block for non-protocol note; got:\n%s", out)
	}
}

// Assertion: TestShowProtocolJSONNoDerivation — --json output does NOT include the derivation text.
func TestShowProtocolJSONNoDerivation(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "My Protocol", note.TypeProtocol)
	n.Body = "Do the thing before acting."
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", n.ID, "--json")
	if err != nil {
		t.Fatalf("nn show --json: %v", err)
	}
	if strings.Contains(out, "## Protocols") {
		t.Errorf("expected no derivation block in JSON output; got:\n%s", out)
	}
}

// Assertion: TestShowGlobalFlag — nn show --global prints all global protocol notes.
func TestShowGlobalFlag(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	p1 := newTestNoteForCLI(note.GenerateID(), "Protocol One", note.TypeProtocol)
	p2 := newTestNoteForCLI(note.GenerateID(), "Protocol Two", note.TypeProtocol)
	writeNoteFile(t, nbDir, p1)
	writeNoteFile(t, nbDir, p2)

	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "Protocol One") {
		t.Errorf("expected 'Protocol One' in output; got:\n%s", out)
	}
	if !strings.Contains(out, "Protocol Two") {
		t.Errorf("expected 'Protocol Two' in output; got:\n%s", out)
	}
}

// Assertion: TestShowGlobalEmpty — nn show --global with no notebook protocols still outputs virtual protocols.
func TestShowGlobalEmpty(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global with no protocols: %v", err)
	}
	if !strings.Contains(out, "virtual-nn-capture-discipline") {
		t.Errorf("expected virtual protocol in output even with empty notebook; got:\n%s", out)
	}
}

// Assertion: TestShowGlobalSeparator — multiple protocols are separated by ---.
func TestShowGlobalSeparator(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	p1 := newTestNoteForCLI(note.GenerateID(), "Protocol One", note.TypeProtocol)
	p2 := newTestNoteForCLI(note.GenerateID(), "Protocol Two", note.TypeProtocol)
	writeNoteFile(t, nbDir, p1)
	writeNoteFile(t, nbDir, p2)

	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, "\n---\n") {
		t.Errorf("expected '---' separator between protocols; got:\n%s", out)
	}
}

func TestShowAppendsToAccessLog(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	cfgDir := t.TempDir()
	t.Setenv("NN_CONFIG_DIR", cfgDir)

	n := newTestNoteForCLI(note.GenerateID(), "Access Me", note.TypeConcept)
	writeNoteFile(t, nbDir, n)

	_, err := execute("show", n.ID)
	if err != nil {
		t.Fatalf("nn show: %v", err)
	}

	logPath := filepath.Join(cfgDir, "access.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("access.log not created: %v", err)
	}
	if !strings.Contains(string(data), n.ID) {
		t.Errorf("access.log %q does not contain note ID %s", string(data), n.ID)
	}
}

// Assertion: TestShowGlobalShowsTodayWhenPresent — nn show --global shows today's daily note even when it already exists.
func TestShowGlobalShowsTodayWhenPresent(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	today := time.Now().Format("2006-01-02")
	todayTitle := "Daily: " + today
	n := newTestNoteForCLI(note.GenerateID(), todayTitle, note.TypeObservation)
	n.Tags = []string{"daily"}
	n.Status = note.StatusPermanent
	n.Body = "## Done\n- existing work"
	writeNoteFile(t, nbDir, n)

	out, err := execute("show", "--global")
	if err != nil {
		t.Fatalf("nn show --global: %v", err)
	}
	if !strings.Contains(out, todayTitle) {
		t.Errorf("nn show --global: want today note %q shown even when already present, got:\n%s", todayTitle, out)
	}
}
