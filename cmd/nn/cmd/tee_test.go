package cmd

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestTeeCommandRegistered(t *testing.T) {
	state := &rootState{}
	cmd := newTeeCmd(state)
	if cmd.Use != "tee" {
		t.Fatalf("expected Use=tee, got %q", cmd.Use)
	}
}

func TestTeePassthrough(t *testing.T) {
	// stdout contains exactly the bytes read from stdin
	input := "hello world\nsome content\n"
	var stdout, stderr bytes.Buffer
	err := runTee(strings.NewReader(input), &stdout, &stderr, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if stdout.String() != input {
		t.Fatalf("expected stdout %q, got %q", input, stdout.String())
	}
}

func TestTeeRelatedToStderr(t *testing.T) {
	// Related notes: printed to stderr, not stdout
	input := "hello world"
	var stdout, stderr bytes.Buffer
	_ = runTee(strings.NewReader(input), &stdout, &stderr, nil, nil)
	// stdout must not contain "Related notes:"
	if strings.Contains(stdout.String(), "Related notes:") {
		t.Fatal("Related notes: must not appear on stdout")
	}
}

func TestTeeExitCode(t *testing.T) {
	// exits 0 when stdin processes without error
	var stdout, stderr bytes.Buffer
	err := runTee(strings.NewReader("ok"), &stdout, &stderr, nil, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	// nonzero when stdin returns error
	errReader := errorReader{err: errors.New("stdin error")}
	err = runTee(errReader, &stdout, &stderr, nil, nil)
	if err == nil {
		t.Fatal("expected error from failing stdin")
	}
}

type errorReader struct{ err error }

func (e errorReader) Read(_ []byte) (int, error) { return 0, e.err }

// ensure runTee signature exists
var _ = runTee

// ensure io.Reader is satisfied
var _ io.Reader = errorReader{}
