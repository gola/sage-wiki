// Package parsers provides lightweight code parsers for extracting structural
// information (imports, exports, types, functions) from source files.
// Each parser targets ~90% coverage of common patterns via regex.
// Graceful degradation: misses symbols rather than hallucinating.
package parsers

import (
	"path/filepath"
	"strings"
)

// Parser extracts structural information from a source file.
type Parser interface {
	// Parse extracts structural information from source content.
	Parse(path string, content []byte) (*ParseResult, error)
	// Extensions returns file extensions this parser handles (without dot).
	Extensions() []string
}

// ParseResult holds extracted structural information.
type ParseResult struct {
	Language  string
	Imports   []Import
	Exports   []Export
	Types     []TypeDecl
	Functions []FuncDecl
	Constants []string
	Structure string // human-readable structural summary
}

// Import represents an import/require/use statement.
type Import struct {
	Path  string
	Alias string
}

// Export represents an exported symbol.
type Export struct {
	Name string
	Kind string // "function", "class", "type", "const", "var", "interface"
}

// TypeDecl represents a type declaration (struct, class, interface, enum).
type TypeDecl struct {
	Name     string
	Kind     string // "struct", "interface", "class", "enum", "type"
	Fields   []string
	Methods  []string
	Exported bool
}

// FuncDecl represents a function or method declaration.
type FuncDecl struct {
	Name      string
	Signature string
	Receiver  string // Go receiver, empty for others
	Exported  bool
}

// Registry maps file extensions to parsers.
type Registry struct {
	parsers map[string]Parser
}

// NewRegistry creates a registry with all built-in parsers.
func NewRegistry() *Registry {
	r := &Registry{parsers: make(map[string]Parser)}
	r.register(&GoParser{})
	r.register(&TypeScriptParser{})
	r.register(&PythonParser{})
	r.register(&RustParser{})
	r.register(&JavaParser{})
	r.register(&CParser{})
	r.register(&RubyParser{})
	r.register(&JSONParser{})
	r.register(&YAMLParser{})
	r.register(&TOMLParser{})
	return r
}

func (r *Registry) register(p Parser) {
	for _, ext := range p.Extensions() {
		r.parsers[ext] = p
	}
}

// Parse runs the appropriate parser for the given file path.
// Returns nil, nil if no parser is registered for the file type.
func (r *Registry) Parse(path string, content []byte) (*ParseResult, error) {
	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	p, ok := r.parsers[ext]
	if !ok {
		return nil, nil
	}
	return p.Parse(path, content)
}

// Supports returns true if a parser exists for the given file extension.
func (r *Registry) Supports(ext string) bool {
	_, ok := r.parsers[strings.TrimPrefix(ext, ".")]
	return ok
}

// FormatStructure builds a human-readable structural summary from a ParseResult.
func FormatStructure(r *ParseResult) string {
	if r == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString("Language: " + r.Language + "\n")

	if len(r.Imports) > 0 {
		b.WriteString("\nImports:\n")
		for _, imp := range r.Imports {
			if imp.Alias != "" {
				b.WriteString("  " + imp.Alias + " = " + imp.Path + "\n")
			} else {
				b.WriteString("  " + imp.Path + "\n")
			}
		}
	}

	if len(r.Types) > 0 {
		b.WriteString("\nTypes:\n")
		for _, t := range r.Types {
			b.WriteString("  " + t.Kind + " " + t.Name)
			if len(t.Fields) > 0 {
				b.WriteString(" { " + strings.Join(t.Fields, ", ") + " }")
			}
			b.WriteString("\n")
		}
	}

	if len(r.Functions) > 0 {
		b.WriteString("\nFunctions:\n")
		for _, f := range r.Functions {
			b.WriteString("  " + f.Signature + "\n")
		}
	}

	if len(r.Exports) > 0 {
		b.WriteString("\nExports:\n")
		for _, e := range r.Exports {
			b.WriteString("  " + e.Kind + " " + e.Name + "\n")
		}
	}

	if len(r.Constants) > 0 {
		b.WriteString("\nConstants: " + strings.Join(r.Constants, ", ") + "\n")
	}

	return b.String()
}
