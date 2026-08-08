package ast_test

import (
	"os"
	"path/filepath"
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
	if helloSym.Body == "" {
		t.Errorf("FAIL: TestSymbolBodyPopulated: expected Symbol.Body to be populated with function source, got empty string")
	}
	if helloSym.Body != "" && len(helloSym.Body) <= len("func HelloWorld()") {
		t.Errorf("FAIL: TestSymbolBodyPopulated: Symbol.Body too short to contain function body, got %q", helloSym.Body)
	}
}
