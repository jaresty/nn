package ast_test

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jaresty/nn/internal/ast"
)

func TestMjsParse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "example.mjs")
	if err := os.WriteFile(path, []byte(`export function greet(name) {
	return "hello " + name;
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := ast.Parse(path)
	if err != nil {
		t.Fatalf("FAIL: TestMjsParse: Parse error: %v", err)
	}
	if f.Language != "javascript" {
		t.Errorf("FAIL: TestMjsParse: expected Language=javascript, got %q", f.Language)
	}
	var found bool
	for _, sym := range f.Symbols {
		if sym.Name == "greet" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("FAIL: TestMjsParse: expected function symbol 'greet' to be extracted from .mjs file")
	}
}

func TestMarkdownParse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "guide.md")
	content := `# Overview
Intro text.

Setup
-----
Install it.

~~~markdown
# Not a heading
Fake
====
~~~

## Usage ##
Run it.

## C#
Language notes.
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := ast.Parse(path)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if f.Language != "markdown" {
		t.Fatalf("Language = %q, want markdown", f.Language)
	}

	var names []string
	for _, sym := range f.Symbols {
		names = append(names, sym.Name)
		if sym.Kind != "heading" {
			t.Errorf("symbol %q kind = %q, want heading", sym.Name, sym.Kind)
		}
		if sym.Body == "" {
			t.Errorf("symbol %q has empty body", sym.Name)
		}
	}
	want := []string{"Overview", "Setup", "Usage", "C#"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("heading names = %#v, want %#v", names, want)
	}
	if got := f.Symbols[0]; got.Line != 1 || got.Column != 3 || got.Signature != "# Overview" {
		t.Errorf("Overview symbol = %#v", got)
	}
	if got := f.Symbols[1]; got.Line != 4 || got.Column != 1 || got.Signature != "Setup" {
		t.Errorf("Setup symbol = %#v", got)
	}
}

func TestMarkdownLongExtension(t *testing.T) {
	path := filepath.Join(t.TempDir(), "guide.markdown")
	if err := os.WriteFile(path, []byte("# Guide\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := ast.Parse(path)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(f.Symbols) != 1 || f.Symbols[0].Name != "Guide" {
		t.Fatalf("symbols = %#v, want Guide heading", f.Symbols)
	}
}

func TestSymbolBodyPopulated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "example.go")
	if err := os.WriteFile(path, []byte(`package main

func HelloWorld() {
	// distinctivebodytermabc
	_ = 42
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	f, err := ast.Parse(path)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	var helloSym *ast.Symbol
	for i := range f.Symbols {
		if f.Symbols[i].Name == "HelloWorld" {
			helloSym = &f.Symbols[i]
			break
		}
	}
	if helloSym == nil {
		t.Fatal("HelloWorld symbol not found")
	}
	if helloSym.Line != 3 || helloSym.Column != 6 {
		t.Errorf("symbol position = %d:%d, want 3:6", helloSym.Line, helloSym.Column)
	}
	if helloSym.Body == "" {
		t.Errorf("FAIL: TestSymbolBodyPopulated: expected Symbol.Body to be populated with function source, got empty string")
	}
	if helloSym.Body != "" && len(helloSym.Body) <= len("func HelloWorld()") {
		t.Errorf("FAIL: TestSymbolBodyPopulated: Symbol.Body too short to contain function body, got %q", helloSym.Body)
	}
}
