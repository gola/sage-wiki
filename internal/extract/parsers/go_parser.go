package parsers

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// GoParser uses Go's stdlib go/parser + go/ast for perfect accuracy.
type GoParser struct{}

func (p *GoParser) Extensions() []string { return []string{"go"} }

func (p *GoParser) Parse(path string, content []byte) (*ParseResult, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("go parse: %w", err)
	}

	result := &ParseResult{Language: "go"}

	// Package
	if f.Name != nil {
		result.Constants = append(result.Constants, "package:"+f.Name.Name)
	}

	// Imports
	for _, imp := range f.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		alias := ""
		if imp.Name != nil {
			alias = imp.Name.Name
		}
		result.Imports = append(result.Imports, Import{Path: path, Alias: alias})
	}

	// Declarations
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					td := TypeDecl{
						Name:     s.Name.Name,
						Exported: s.Name.IsExported(),
					}
					switch st := s.Type.(type) {
					case *ast.StructType:
						td.Kind = "struct"
						if st.Fields != nil {
							for _, field := range st.Fields.List {
								for _, name := range field.Names {
									td.Fields = append(td.Fields, name.Name)
								}
							}
						}
					case *ast.InterfaceType:
						td.Kind = "interface"
						if st.Methods != nil {
							for _, method := range st.Methods.List {
								for _, name := range method.Names {
									td.Methods = append(td.Methods, name.Name)
								}
							}
						}
					default:
						td.Kind = "type"
					}
					result.Types = append(result.Types, td)
					if td.Exported {
						result.Exports = append(result.Exports, Export{Name: td.Name, Kind: td.Kind})
					}

				case *ast.ValueSpec:
					for _, name := range s.Names {
						if name.IsExported() {
							kind := "var"
							if d.Tok == token.CONST {
								kind = "const"
								result.Constants = append(result.Constants, name.Name)
							}
							result.Exports = append(result.Exports, Export{Name: name.Name, Kind: kind})
						} else if d.Tok == token.CONST {
							result.Constants = append(result.Constants, name.Name)
						}
					}
				}
			}

		case *ast.FuncDecl:
			fd := FuncDecl{
				Name:     d.Name.Name,
				Exported: d.Name.IsExported(),
			}

			// Build signature
			var sig strings.Builder
			sig.WriteString("func ")
			if d.Recv != nil && len(d.Recv.List) > 0 {
				recv := d.Recv.List[0]
				recvType := formatGoType(recv.Type)
				fd.Receiver = recvType
				if len(recv.Names) > 0 {
					sig.WriteString(fmt.Sprintf("(%s %s) ", recv.Names[0].Name, recvType))
				} else {
					sig.WriteString(fmt.Sprintf("(%s) ", recvType))
				}
			}
			sig.WriteString(d.Name.Name)
			sig.WriteString(formatGoParams(d.Type.Params))
			if d.Type.Results != nil && len(d.Type.Results.List) > 0 {
				sig.WriteString(" ")
				sig.WriteString(formatGoParams(d.Type.Results))
			}
			fd.Signature = sig.String()

			result.Functions = append(result.Functions, fd)
			if fd.Exported {
				result.Exports = append(result.Exports, Export{Name: fd.Name, Kind: "function"})
			}
		}
	}

	result.Structure = FormatStructure(result)
	return result, nil
}

func formatGoType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + formatGoType(t.X)
	case *ast.SelectorExpr:
		return formatGoType(t.X) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + formatGoType(t.Elt)
	case *ast.MapType:
		return "map[" + formatGoType(t.Key) + "]" + formatGoType(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.FuncType:
		return "func(...)"
	default:
		return "..."
	}
}

func formatGoParams(fl *ast.FieldList) string {
	if fl == nil || len(fl.List) == 0 {
		return "()"
	}
	var parts []string
	for _, f := range fl.List {
		typeName := formatGoType(f.Type)
		if len(f.Names) > 0 {
			for _, n := range f.Names {
				parts = append(parts, n.Name+" "+typeName)
			}
		} else {
			parts = append(parts, typeName)
		}
	}
	return "(" + strings.Join(parts, ", ") + ")"
}
