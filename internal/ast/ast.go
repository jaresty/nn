// Package ast provides a lightweight structural outline of source files for LLM consumption.
// It uses gotreesitter (pure Go, no CGo) to parse source files and extract symbols.
// This package is isolated so the gotreesitter dependency can be replaced without
// affecting the rest of the codebase.
package ast

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/odvcencio/gotreesitter"
	"github.com/odvcencio/gotreesitter/grammars"
)

// Symbol is a structural element extracted from a source file.
type Symbol struct {
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	Signature string `json:"signature"`
	Line      int    `json:"line"` // 1-indexed
}

// File holds the parsed outline of a source file.
type File struct {
	Path     string
	Language string
	Symbols  []Symbol
}

// langConfig holds the queries for a specific language.
type langConfig struct {
	name        string
	importQuery string // S-expr query; capture name @import
	symbolQuery string // S-expr query; each capture is a symbol with @kind and @name
}

// per-language query configurations.
var langConfigs = map[string]langConfig{
	".go": {
		name:        "go",
		importQuery: `(import_spec path: (interpreted_string_literal) @import)`,
		symbolQuery: `
(function_declaration name: (identifier) @name) @function
(method_declaration name: (field_identifier) @name) @method
(type_declaration (type_spec name: (type_identifier) @name)) @type
(const_declaration (const_spec name: (identifier) @name)) @const
`,
	},
	".py": {
		name:        "python",
		importQuery: `(import_statement name: (dotted_name) @import)`,
		symbolQuery: `
(function_definition name: (identifier) @name) @function
(class_definition name: (identifier) @name) @class
`,
	},
	".js": {
		name:        "javascript",
		importQuery: `(import_statement source: (string) @import)`,
		symbolQuery: `
(function_declaration name: (identifier) @name) @function
(class_declaration name: (identifier) @name) @class
(variable_declarator name: (identifier) @name) @variable
`,
	},
	".ts": {
		name:        "typescript",
		importQuery: `(import_statement source: (string) @import)`,
		symbolQuery: `
(function_declaration name: (identifier) @name) @function
(class_declaration name: (identifier) @name) @class
(interface_declaration name: (type_identifier) @name) @interface
(type_alias_declaration name: (type_identifier) @name) @type
`,
	},
	".rs": {
		name:        "rust",
		importQuery: `(use_declaration argument: (_) @import)`,
		symbolQuery: `
(function_item name: (identifier) @name) @function
(struct_item name: (type_identifier) @name) @struct
(enum_item name: (type_identifier) @name) @enum
(trait_item name: (type_identifier) @name) @trait
(impl_item type: (type_identifier) @name) @impl
`,
	},
	".java": {
		name:        "java",
		importQuery: `(import_declaration (scoped_identifier) @import)`,
		symbolQuery: `
(method_declaration name: (identifier) @name) @method
(class_declaration name: (identifier) @name) @class
(interface_declaration name: (identifier) @name) @interface
`,
	},
}

// Parse reads filePath, detects its language, and returns a structural outline.
func Parse(filePath string) (*File, error) {
	ext := strings.ToLower(filepath.Ext(filePath))
	cfg, ok := langConfigs[ext]
	if !ok {
		// Try gotreesitter language detection as fallback.
		return nil, fmt.Errorf("ast: unsupported file type %q", ext)
	}

	entry := grammars.DetectLanguage(filePath)
	if entry == nil {
		return nil, fmt.Errorf("ast: gotreesitter does not support %q", ext)
	}

	src, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("ast: read %s: %w", filePath, err)
	}

	lang := entry.Language()
	parser := gotreesitter.NewParser(lang)
	tree, err := parser.Parse(src)
	if err != nil {
		return nil, fmt.Errorf("ast: parse %s: %w", filePath, err)
	}

	root := tree.RootNode()

	// Extract imports.
	var imports []string
	if cfg.importQuery != "" {
		imports = runQuery(lang, root, src, cfg.importQuery, "import")
		// For Go: strip quotes and use the last path component.
		if ext == ".go" {
			cleaned := make([]string, 0, len(imports))
			for _, imp := range imports {
				imp = strings.Trim(imp, `"`)
				parts := strings.Split(imp, "/")
				cleaned = append(cleaned, parts[len(parts)-1])
			}
			imports = cleaned
		}
	}

	// Extract symbols.
	var symbols []Symbol
	if imports != nil {
		symbols = append(symbols, Symbol{
			Kind:      "import",
			Name:      strings.Join(imports, ", "),
			Signature: "imports: " + strings.Join(imports, ", "),
			Line:      0,
		})
	}

	if cfg.symbolQuery != "" {
		lines := strings.Split(string(src), "\n")
		syms := extractSymbols(lang, root, src, cfg.symbolQuery, lines)
		symbols = append(symbols, syms...)
	}

	return &File{
		Path:     filePath,
		Language: cfg.name,
		Symbols:  symbols,
	}, nil
}

// runQuery executes a query and returns all text captures named captureName.
func runQuery(lang *gotreesitter.Language, root *gotreesitter.Node, src []byte, queryStr, captureName string) []string {
	q, err := gotreesitter.NewQuery(queryStr, lang)
	if err != nil {
		return nil
	}
	cursor := q.Exec(root, lang, src)
	var results []string
	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}
		for _, cap := range match.Captures {
			results = append(results, cap.Node.Text(src))
		}
	}
	return results
}

// extractSymbols uses a symbol query to build Symbol entries.
// The query must have an @name capture. The top-level node type becomes the kind.
func extractSymbols(lang *gotreesitter.Language, root *gotreesitter.Node, src []byte, queryStr string, lines []string) []Symbol {
	q, err := gotreesitter.NewQuery(queryStr, lang)
	if err != nil {
		return nil
	}
	cursor := q.Exec(root, lang, src)
	var results []Symbol
	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}
		var kind, name string
		var line uint32
		// The first non-@name capture gives us the kind; @name gives name and line.
		for _, cap := range match.Captures {
			if cap.Name == "name" {
				name = cap.Node.Text(src)
				line = cap.Node.StartPoint().Row
			} else {
				// The capture name is the kind (function, method, type, etc.)
				kind = cap.Name
			}
		}
		if name == "" {
			continue
		}
		sig := ""
		if int(line) < len(lines) {
			sig = strings.TrimSpace(lines[line])
		}
		results = append(results, Symbol{
			Kind:      kind,
			Name:      name,
			Signature: sig,
			Line:      int(line) + 1,
		})
	}
	return results
}
