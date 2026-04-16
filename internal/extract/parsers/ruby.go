package parsers

import "regexp"

// RubyParser handles .rb files via regex.
type RubyParser struct{}

func (p *RubyParser) Extensions() []string { return []string{"rb"} }

var (
	rbRequireRe = regexp.MustCompile(`(?m)^require\s+['"]([^'"]+)['"]`)
	rbClassRe   = regexp.MustCompile(`(?m)^class\s+(\w+)`)
	rbModuleRe  = regexp.MustCompile(`(?m)^module\s+(\w+)`)
	rbDefRe     = regexp.MustCompile(`(?m)^\s*def\s+(self\.)?(\w+[?!=]?)\s*(?:\(([^)]*)\))?`)
	rbAttrRe    = regexp.MustCompile(`(?m)^\s*attr_(?:accessor|reader|writer)\s+(.+)`)
)

func (p *RubyParser) Parse(path string, content []byte) (*ParseResult, error) {
	src := string(content)
	result := &ParseResult{Language: "ruby"}

	for _, m := range rbRequireRe.FindAllStringSubmatch(src, -1) {
		result.Imports = append(result.Imports, Import{Path: m[1]})
	}

	for _, m := range rbClassRe.FindAllStringSubmatch(src, -1) {
		result.Types = append(result.Types, TypeDecl{Name: m[1], Kind: "class", Exported: true})
		result.Exports = append(result.Exports, Export{Name: m[1], Kind: "class"})
	}
	for _, m := range rbModuleRe.FindAllStringSubmatch(src, -1) {
		result.Constants = append(result.Constants, "module:"+m[1])
	}

	for _, m := range rbDefRe.FindAllStringSubmatch(src, -1) {
		name := m[2]
		isSelf := m[1] != ""
		sig := "def " + name
		if m[3] != "" {
			sig += "(" + m[3] + ")"
		}
		result.Functions = append(result.Functions, FuncDecl{
			Name: name, Signature: sig, Exported: !isSelf && name[0] != '_',
		})
	}

	for _, m := range rbAttrRe.FindAllStringSubmatch(src, -1) {
		result.Constants = append(result.Constants, "attr:"+m[1])
	}

	result.Structure = FormatStructure(result)
	return result, nil
}
