package parsers

import "regexp"

// JavaParser handles .java files via regex.
type JavaParser struct{}

func (p *JavaParser) Extensions() []string { return []string{"java"} }

var (
	javaImportRe    = regexp.MustCompile(`(?m)^import\s+(?:static\s+)?([^;]+);`)
	javaClassRe     = regexp.MustCompile(`(?m)^(?:public\s+)?(?:abstract\s+)?(?:final\s+)?class\s+(\w+)`)
	javaInterfaceRe = regexp.MustCompile(`(?m)^(?:public\s+)?interface\s+(\w+)`)
	javaEnumRe      = regexp.MustCompile(`(?m)^(?:public\s+)?enum\s+(\w+)`)
	javaMethodRe    = regexp.MustCompile(`(?m)^\s+(?:public|protected|private)\s+(?:static\s+)?(?:abstract\s+)?(?:final\s+)?(?:synchronized\s+)?(?:\w+(?:<[^>]*>)?)\s+(\w+)\s*\(([^)]*)\)`)
	javaAnnotRe     = regexp.MustCompile(`(?m)^@(\w+)`)
)

func (p *JavaParser) Parse(path string, content []byte) (*ParseResult, error) {
	src := string(content)
	result := &ParseResult{Language: "java"}

	for _, m := range javaImportRe.FindAllStringSubmatch(src, -1) {
		result.Imports = append(result.Imports, Import{Path: m[1]})
	}

	for _, m := range javaClassRe.FindAllStringSubmatch(src, -1) {
		result.Types = append(result.Types, TypeDecl{Name: m[1], Kind: "class", Exported: true})
		result.Exports = append(result.Exports, Export{Name: m[1], Kind: "class"})
	}
	for _, m := range javaInterfaceRe.FindAllStringSubmatch(src, -1) {
		result.Types = append(result.Types, TypeDecl{Name: m[1], Kind: "interface", Exported: true})
		result.Exports = append(result.Exports, Export{Name: m[1], Kind: "interface"})
	}
	for _, m := range javaEnumRe.FindAllStringSubmatch(src, -1) {
		result.Types = append(result.Types, TypeDecl{Name: m[1], Kind: "enum", Exported: true})
		result.Exports = append(result.Exports, Export{Name: m[1], Kind: "enum"})
	}

	for _, m := range javaMethodRe.FindAllStringSubmatch(src, -1) {
		result.Functions = append(result.Functions, FuncDecl{
			Name: m[1], Signature: m[0], Exported: true,
		})
	}

	for _, m := range javaAnnotRe.FindAllStringSubmatch(src, -1) {
		result.Constants = append(result.Constants, "@"+m[1])
	}

	result.Structure = FormatStructure(result)
	return result, nil
}
