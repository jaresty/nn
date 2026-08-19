package cmd

import (
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/jaresty/nn/internal/note"
	"github.com/jaresty/nn/internal/rules"
)

func engineBlockedSetReference(tb testing.TB, notes []*note.Note) map[string]bool {
	tb.Helper()
	e := rules.NewEngine()
	for _, f := range rules.FactsFromNotes(notes) {
		e.AddFact(f)
	}
	builtin, err := rules.ParseProgram(rules.BuiltinRules())
	if err != nil {
		tb.Fatalf("parse builtins: %v", err)
	}
	e.AddRules(builtin)
	if err := e.Eval(); err != nil {
		tb.Fatalf("evaluate builtins: %v", err)
	}
	got := map[string]bool{}
	for _, f := range e.Query("blocked") {
		if len(f.Args) > 0 {
			got[f.Args[0]] = true
		}
	}
	return got
}

func TestBlockedSetMatchesRulesEngine(t *testing.T) {
	for seed := int64(0); seed < 25; seed++ {
		rng := rand.New(rand.NewSource(seed))
		notes := make([]*note.Note, 20)
		for i := range notes {
			body := "done"
			if rng.Intn(3) == 0 {
				body = "- [ ] open"
			}
			notes[i] = &note.Note{ID: fmt.Sprintf("n%d", i), Type: note.TypeConcept, Body: body}
		}
		for i := range notes {
			for range rng.Intn(3) {
				target := rng.Intn(len(notes))
				notes[i].Links = append(notes[i].Links, note.Link{TargetID: notes[target].ID, Type: "requires"})
			}
		}
		if got, want := blockedSet(notes), engineBlockedSetReference(t, notes); !reflect.DeepEqual(got, want) {
			t.Fatalf("seed %d: blockedSet=%v, engine=%v", seed, got, want)
		}
	}
}

func TestBlockedSetUsesPredicateDirectedRulesEvaluation(t *testing.T) {
	source, err := os.ReadFile("todo.go")
	if err != nil {
		t.Fatalf("read todo.go: %v", err)
	}
	start := strings.Index(string(source), "func blockedSet(")
	if start < 0 {
		t.Fatal("func blockedSet not found")
	}
	body := string(source[start:])
	if !strings.Contains(body, `EvalFor("blocked")`) {
		t.Fatal(`blockedSet must use EvalFor("blocked")`)
	}
	if strings.Contains(body, ".Eval()") {
		t.Fatal("blockedSet must not evaluate the complete ruleset")
	}
}

func blockedBenchmarkNotes(n, unrelatedEdges, requiresEdges int) []*note.Note {
	notes := make([]*note.Note, n)
	for i := range notes {
		notes[i] = &note.Note{ID: fmt.Sprintf("n%d", i), Type: note.TypeConcept, Body: "done"}
	}
	for i := 0; i < unrelatedEdges; i++ {
		from := i % n
		to := (i*17 + 1) % n
		notes[from].Links = append(notes[from].Links, note.Link{TargetID: notes[to].ID, Type: "supports"})
	}
	for i := 0; i < requiresEdges; i++ {
		from := i % (n - 1)
		to := from + 1
		notes[from].Links = append(notes[from].Links, note.Link{TargetID: notes[to].ID, Type: "requires"})
	}
	notes[n-1].Body = "- [ ] open"
	return notes
}

func BenchmarkBlockedSetTopology(b *testing.B) {
	notes := blockedBenchmarkNotes(1000, 2500, 20)
	b.Run("directed-engine", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			blockedSet(notes)
		}
	})
	b.Run("general-engine", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			engineBlockedSetReference(b, notes)
		}
	})
}

func BenchmarkBlockedSetLive(b *testing.B) {
	if os.Getenv("NN_BENCH_LIVE") != "1" {
		b.Skip("set NN_BENCH_LIVE=1 to benchmark the configured live notebook")
	}
	state := &rootState{}
	cmd := &cobra.Command{Use: "benchmark-blocked-set"}
	if err := initState(cmd, state, ""); err != nil {
		b.Fatal(err)
	}
	notes, err := state.backend.List()
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		blockedSet(notes)
	}
}
