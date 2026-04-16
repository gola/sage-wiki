package parsers

import (
	"regexp"
	"strings"
)

// TypeScriptParser handles .ts, .tsx, .js, .jsx files via regex.
type TypeScriptParser struct{}

func (p *TypeScriptParser) Extensions() []string { return []string{"ts", "tsx", "js", "jsx"} }

var (
	tsImportRe    = regexp.MustCompile(`(?m)^import\s+(?:{[^}]+}|\*\s+as\s+\w+|\w+)\s+from\s+['"]([^'"]+)['"]`)
	tsRequireRe   = regexp.MustCompile(`(?m)(?:const|let|var)\s+(\w+)\s*=\s*require\(['"]([^'"]+)['"]\)`)
	tsExportFnRe  = regexp.MustCompile(`(?m)^export\s+(?:async\s+)?function\s+(\w+)`)
	tsExportClRe  = regexp.MustCompile(`(?m)^export\s+(?:default\s+)?class\s+(\w+)`)
	tsClassRe     = regexp.MustCompile(`(?m)^(?:export\s+)?(?:abstract\s+)?class\s+(\w+)`)
	tsInterfaceRe = regexp.MustCompile(`(?m)^(?:export\s+)?interface\s+(\w+)`)
	tsTypeRe      = regexp.MustCompile(`(?m)^(?:export\s+)?type\s+(\w+)`)
	tsFuncRe      = regexp.MustCompile(`(?m)^(?:export\s+)?(?:async\s+)?function\s+(\w+)\s*(?:<[^>]*>)?\s*\(([^)]*)\)`)
	tsConstRe     = regexp.MustCompile(`(?m)^export\s+const\s+(\w+)`)
	tsEnumRe      = regexp.MustCompile(`(?m)^(?:export\s+)?enum\s+(\w+)`)
)

func (p *TypeScriptParser) Parse(path string, content []byte) (*ParseResult, error) {
	src := string(content)
	lang := "typescript"
	ext := strings.TrimPrefix(path[strings.LastIndex(path, "."):], ".")
	if ext == "js" || ext == "jsx" {
		lang = "javascript"
	}
	result := &ParseResult{Language: lang}

	// Imports
	for _, m := range tsImportRe.FindAllStringSubmatch(src, -1) {
		result.Imports = append(result.Imports, Import{Path: m[1]})
	}
	for _, m := range tsRequireRe.FindAllStringSubmatch(src, -1) {
		result.Imports = append(result.Imports, Import{Path: m[2], Alias: m[1]})
	}

	// Classes
	for _, m := range tsClassRe.FindAllStringSubmatch(src, -1) {
		result.Types = append(result.Types, TypeDecl{Name: m[1], Kind: "class", Exported: true})
	}

	// Interfaces
	for _, m := range tsInterfaceRe.FindAllStringSubmatch(src, -1) {
		result.Types = append(result.Types, TypeDecl{Name: m[1], Kind: "interface", Exported: true})
	}

	// Type aliases
	for _, m := range tsTypeRe.FindAllStringSubmatch(src, -1) {
		result.Types = append(result.Types, TypeDecl{Name: m[1], Kind: "type", Exported: true})
	}

	// Enums
	for _, m := range tsEnumRe.FindAllStringSubmatch(src, -1) {
		result.Types = append(result.Types, TypeDecl{Name: m[1], Kind: "enum", Exported: true})
	}

	// Functions
	for _, m := range tsFuncRe.FindAllStringSubmatch(src, -1) {
		exported := strings.Contains(m[0], "export")
		result.Functions = append(result.Functions, FuncDecl{
			Name:      m[1],
			Signature: strings.TrimSpace(m[0]),
			Exported:  exported,
		})
	}

	// Exported functions (from export keyword)
	for _, m := range tsExportFnRe.FindAllStringSubmatch(src, -1) {
		result.Exports = append(result.Exports, Export{Name: m[1], Kind: "function"})
	}
	for _, m := range tsExportClRe.FindAllStringSubmatch(src, -1) {
		result.Exports = append(result.Exports, Export{Name: m[1], Kind: "class"})
	}
	for _, m := range tsConstRe.FindAllStringSubmatch(src, -1) {
		result.Exports = append(result.Exports, Export{Name: m[1], Kind: "const"})
	}

	result.Structure = FormatStructure(result)
	return result, nil
}
