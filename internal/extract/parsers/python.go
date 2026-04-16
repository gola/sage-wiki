package parsers

import (
	"regexp"
)

// PythonParser handles .py files via regex.
type PythonParser struct{}

func (p *PythonParser) Extensions() []string { return []string{"py"} }

var (
	pyImportRe    = regexp.MustCompile(`(?m)^import\s+(\S+)`)
	pyFromRe      = regexp.MustCompile(`(?m)^from\s+(\S+)\s+import`)
	pyClassRe     = regexp.MustCompile(`(?m)^class\s+(\w+)`)
	pyFuncRe      = regexp.MustCompile(`(?m)^(\s*)def\s+(\w+)\s*\(([^)]*)\)`)
	pyAsyncFuncRe = regexp.MustCompile(`(?m)^(\s*)async\s+def\s+(\w+)\s*\(([^)]*)\)`)
	pyDecoratorRe = regexp.MustCompile(`(?m)^\s*@(\w+)`)
)

func (p *PythonParser) Parse(path string, content []byte) (*ParseResult, error) {
	src := string(content)
	result := &ParseResult{Language: "python"}

	for _, m := range pyImportRe.FindAllStringSubmatch(src, -1) {
		result.Imports = append(result.Imports, Import{Path: m[1]})
	}
	for _, m := range pyFromRe.FindAllStringSubmatch(src, -1) {
		result.Imports = append(result.Imports, Import{Path: m[1]})
	}

	for _, m := range pyClassRe.FindAllStringSubmatch(src, -1) {
		result.Types = append(result.Types, TypeDecl{Name: m[1], Kind: "class", Exported: true})
		result.Exports = append(result.Exports, Export{Name: m[1], Kind: "class"})
	}

	for _, m := range pyFuncRe.FindAllStringSubmatch(src, -1) {
		indent := m[1] // leading whitespace
		name := m[2]
		params := m[3]
		isMethod := len(indent) > 0 // indented = class method
		exported := name[0] != '_'
		result.Functions = append(result.Functions, FuncDecl{
			Name: name, Signature: "def " + name + "(" + params + ")", Exported: exported,
		})
		if exported && !isMethod {
			result.Exports = append(result.Exports, Export{Name: name, Kind: "function"})
		}
	}
	for _, m := range pyAsyncFuncRe.FindAllStringSubmatch(src, -1) {
		indent := m[1]
		name := m[2]
		params := m[3]
		isMethod := len(indent) > 0
		exported := name[0] != '_'
		result.Functions = append(result.Functions, FuncDecl{
			Name: name, Signature: "async def " + name + "(" + params + ")", Exported: exported,
		})
		if exported && !isMethod {
			result.Exports = append(result.Exports, Export{Name: name, Kind: "function"})
		}
	}

	// Count decorators as metadata
	for _, m := range pyDecoratorRe.FindAllStringSubmatch(src, -1) {
		result.Constants = append(result.Constants, "@"+m[1])
	}

	result.Structure = FormatStructure(result)
	return result, nil
}
