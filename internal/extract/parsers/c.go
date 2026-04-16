package parsers

import "regexp"

// CParser handles .c, .cpp, .h, .hpp files via regex.
type CParser struct{}

func (p *CParser) Extensions() []string { return []string{"c", "cpp", "h", "hpp", "cc", "cxx"} }

var (
	cIncludeRe   = regexp.MustCompile(`(?m)^#include\s+[<"]([^>"]+)[>"]`)
	cFuncRe      = regexp.MustCompile(`(?m)^(?:static\s+)?(?:inline\s+)?(?:extern\s+)?(?:const\s+)?(\w[\w*&\s]+?)\s+(\w+)\s*\(([^)]*)\)\s*\{`)
	cStructRe    = regexp.MustCompile(`(?m)^(?:typedef\s+)?struct\s+(\w+)`)
	cClassRe     = regexp.MustCompile(`(?m)^class\s+(\w+)`)
	cTypedefRe   = regexp.MustCompile(`(?m)^typedef\s+.+\s+(\w+)\s*;`)
	cNamespaceRe = regexp.MustCompile(`(?m)^namespace\s+(\w+)`)
	cEnumRe      = regexp.MustCompile(`(?m)^(?:typedef\s+)?enum\s+(\w+)`)
)

func (p *CParser) Parse(path string, content []byte) (*ParseResult, error) {
	src := string(content)
	lang := "c"
	for _, ext := range []string{".cpp", ".hpp", ".cc", ".cxx"} {
		if len(path) > len(ext) && path[len(path)-len(ext):] == ext {
			lang = "cpp"
			break
		}
	}
	result := &ParseResult{Language: lang}

	for _, m := range cIncludeRe.FindAllStringSubmatch(src, -1) {
		result.Imports = append(result.Imports, Import{Path: m[1]})
	}

	for _, m := range cStructRe.FindAllStringSubmatch(src, -1) {
		result.Types = append(result.Types, TypeDecl{Name: m[1], Kind: "struct", Exported: true})
		result.Exports = append(result.Exports, Export{Name: m[1], Kind: "struct"})
	}
	for _, m := range cClassRe.FindAllStringSubmatch(src, -1) {
		result.Types = append(result.Types, TypeDecl{Name: m[1], Kind: "class", Exported: true})
		result.Exports = append(result.Exports, Export{Name: m[1], Kind: "class"})
	}
	for _, m := range cEnumRe.FindAllStringSubmatch(src, -1) {
		result.Types = append(result.Types, TypeDecl{Name: m[1], Kind: "enum", Exported: true})
	}
	for _, m := range cTypedefRe.FindAllStringSubmatch(src, -1) {
		result.Types = append(result.Types, TypeDecl{Name: m[1], Kind: "type", Exported: true})
	}
	for _, m := range cNamespaceRe.FindAllStringSubmatch(src, -1) {
		result.Constants = append(result.Constants, "namespace:"+m[1])
	}

	for _, m := range cFuncRe.FindAllStringSubmatch(src, -1) {
		result.Functions = append(result.Functions, FuncDecl{
			Name: m[2], Signature: m[1] + " " + m[2] + "(" + m[3] + ")", Exported: true,
		})
		result.Exports = append(result.Exports, Export{Name: m[2], Kind: "function"})
	}

	result.Structure = FormatStructure(result)
	return result, nil
}
