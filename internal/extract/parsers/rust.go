package parsers

import "regexp"

// RustParser handles .rs files via regex.
type RustParser struct{}

func (p *RustParser) Extensions() []string { return []string{"rs"} }

var (
	rsUseRe    = regexp.MustCompile(`(?m)^use\s+([^;]+);`)
	rsFnRe     = regexp.MustCompile(`(?m)^(pub\s+)?(?:async\s+)?fn\s+(\w+)\s*(?:<[^>]*>)?\s*\(([^)]*)\)`)
	rsStructRe = regexp.MustCompile(`(?m)^(pub\s+)?struct\s+(\w+)`)
	rsEnumRe   = regexp.MustCompile(`(?m)^(pub\s+)?enum\s+(\w+)`)
	rsTraitRe  = regexp.MustCompile(`(?m)^(pub\s+)?trait\s+(\w+)`)
	rsImplRe   = regexp.MustCompile(`(?m)^impl(?:<[^>]*>)?\s+(\w+)`)
	rsModRe    = regexp.MustCompile(`(?m)^(pub\s+)?mod\s+(\w+)`)
	rsTypeRe   = regexp.MustCompile(`(?m)^(pub\s+)?type\s+(\w+)`)
)

func (p *RustParser) Parse(path string, content []byte) (*ParseResult, error) {
	src := string(content)
	result := &ParseResult{Language: "rust"}

	for _, m := range rsUseRe.FindAllStringSubmatch(src, -1) {
		result.Imports = append(result.Imports, Import{Path: m[1]})
	}

	for _, m := range rsStructRe.FindAllStringSubmatch(src, -1) {
		exported := m[1] != ""
		result.Types = append(result.Types, TypeDecl{Name: m[2], Kind: "struct", Exported: exported})
		if exported {
			result.Exports = append(result.Exports, Export{Name: m[2], Kind: "struct"})
		}
	}
	for _, m := range rsEnumRe.FindAllStringSubmatch(src, -1) {
		exported := m[1] != ""
		result.Types = append(result.Types, TypeDecl{Name: m[2], Kind: "enum", Exported: exported})
		if exported {
			result.Exports = append(result.Exports, Export{Name: m[2], Kind: "enum"})
		}
	}
	for _, m := range rsTraitRe.FindAllStringSubmatch(src, -1) {
		exported := m[1] != ""
		result.Types = append(result.Types, TypeDecl{Name: m[2], Kind: "interface", Exported: exported})
		if exported {
			result.Exports = append(result.Exports, Export{Name: m[2], Kind: "interface"})
		}
	}
	for _, m := range rsTypeRe.FindAllStringSubmatch(src, -1) {
		exported := m[1] != ""
		result.Types = append(result.Types, TypeDecl{Name: m[2], Kind: "type", Exported: exported})
	}
	for _, m := range rsModRe.FindAllStringSubmatch(src, -1) {
		result.Constants = append(result.Constants, "mod:"+m[2])
	}

	for _, m := range rsFnRe.FindAllStringSubmatch(src, -1) {
		exported := m[1] != ""
		result.Functions = append(result.Functions, FuncDecl{
			Name: m[2], Signature: m[0], Exported: exported,
		})
		if exported {
			result.Exports = append(result.Exports, Export{Name: m[2], Kind: "function"})
		}
	}

	result.Structure = FormatStructure(result)
	return result, nil
}
