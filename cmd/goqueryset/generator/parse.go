// Package generator generates typed GORM v2 query sets for structs annotated with a "gen:qs" comment line.
//
// It reads one Go source file syntactically (no type checking), so it works on any package that is not yet
// compilable, and it depends only on the field types it can see in that file.
package generator

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"path"
	"reflect"
	"sort"
	"strings"

	"gorm.io/gorm/schema"
)

// kind is what a field supports in a query.
type kind int

const (
	kindEquality kind = iota // Eq, Ne, In, NotIn
	kindOrdered              // + Gt, Gte, Lt, Lte, ordering
	kindString               // + Like, NotLike
	kindBool                 // Eq, Ne
	kindPointer              // Eq, Ne, IsNull, IsNotNull
)

type field struct {
	Name   string // Go field name
	Column string // database column
	Type   string // Go type as written in the source
	Kind   kind
	// Orderable is true for fields that can be sorted (ordered and string kinds, and pointers to them).
	Orderable bool
}

func (f field) HasIn() bool {
	return f.Kind == kindEquality || f.Kind == kindOrdered || f.Kind == kindString
}
func (f field) HasCompare() bool { return f.Kind == kindOrdered || f.Kind == kindString }
func (f field) HasLike() bool    { return f.Kind == kindString }
func (f field) HasNull() bool    { return f.Kind == kindPointer }

type model struct {
	Name   string
	Fields []field
}

type file struct {
	Package string
	Models  []model
	Imports []string // import lines for packages the generated code uses, beyond its own
}

const marker = "gen:qs"

var naming = schema.NamingStrategy{}

// parse extracts the annotated models of one source file.
func parse(filename string, src []byte) (*file, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// Types declared in this file, to classify named field types by their underlying type.
	declared := map[string]ast.Expr{}
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts := spec.(*ast.TypeSpec)
			declared[ts.Name.Name] = ts.Type
		}
	}

	out := &file{Package: f.Name.Name}
	usedPkgs := map[string]bool{}
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts := spec.(*ast.TypeSpec)
			doc := ts.Doc
			if doc == nil && len(gd.Specs) == 1 {
				doc = gd.Doc
			}
			if !hasMarker(doc) {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				return nil, fmt.Errorf("%s: %s is marked %q but is not a struct", fset.Position(ts.Pos()), ts.Name.Name, marker)
			}
			if ts.TypeParams != nil && len(ts.TypeParams.List) > 0 {
				return nil, fmt.Errorf("%s: %s: generic structs are not supported", fset.Position(ts.Pos()), ts.Name.Name)
			}
			m := model{Name: ts.Name.Name}
			for _, fl := range st.Fields.List {
				if len(fl.Names) == 0 {
					continue // embedded fields are not expanded; declare columns explicitly
				}
				column, skip := gormColumn(fl.Tag)
				if skip {
					continue
				}
				k, orderable, ok := classify(fl.Type, declared)
				if !ok {
					continue // relations, slices, maps and other non-column types
				}
				typ := types.ExprString(fl.Type)
				collectPackages(fl.Type, usedPkgs)
				for _, name := range fl.Names {
					if !name.IsExported() {
						continue
					}
					col := column
					if col == "" {
						col = naming.ColumnName("", name.Name)
					}
					m.Fields = append(m.Fields, field{Name: name.Name, Column: col, Type: typ, Kind: k, Orderable: orderable})
				}
			}
			if len(m.Fields) == 0 {
				return nil, fmt.Errorf("%s: %s has no column fields", fset.Position(ts.Pos()), ts.Name.Name)
			}
			out.Models = append(out.Models, m)
		}
	}
	if len(out.Models) == 0 {
		return nil, fmt.Errorf("%s: no struct is marked with a %q comment line", filename, marker)
	}

	imports, err := resolveImports(f, usedPkgs)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filename, err)
	}
	out.Imports = imports
	return out, nil
}

func hasMarker(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}
	for _, c := range doc.List {
		if strings.TrimSpace(strings.TrimPrefix(c.Text, "//")) == marker {
			return true
		}
	}
	return false
}

// gormColumn reads the gorm struct tag: the explicit column name, and whether the field is not a column at all
// (ignored, or a relation).
func gormColumn(tag *ast.BasicLit) (column string, skip bool) {
	if tag == nil {
		return "", false
	}
	raw := strings.Trim(tag.Value, "`")
	for _, part := range strings.Split(reflect.StructTag(raw).Get("gorm"), ";") {
		key, value, _ := strings.Cut(strings.TrimSpace(part), ":")
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "-":
			return "", true
		case "column":
			column = strings.TrimSpace(value)
		case "foreignkey", "references", "many2many", "polymorphic", "embedded":
			return "", true
		}
	}
	return column, false
}

var orderedBasics = map[string]bool{
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
	"float32": true, "float64": true, "byte": true, "rune": true,
}

// classify decides which query methods a field type supports. ok is false for types that are not columns.
func classify(expr ast.Expr, declared map[string]ast.Expr) (k kind, orderable bool, ok bool) {
	switch t := expr.(type) {
	case *ast.Ident:
		switch {
		case t.Name == "string":
			return kindString, true, true
		case t.Name == "bool":
			return kindBool, false, true
		case orderedBasics[t.Name]:
			return kindOrdered, true, true
		}
		if under, found := declared[t.Name]; found {
			if _, isIdent := under.(*ast.Ident); isIdent {
				return classify(under, nil) // e.g. type Status string
			}
			if sel, isSel := under.(*ast.SelectorExpr); isSel {
				return classify(sel, nil)
			}
			return 0, false, false // structs, slices, maps: relations or composite values
		}
		return kindEquality, false, true // declared in another file of the package
	case *ast.SelectorExpr:
		if x, isIdent := t.X.(*ast.Ident); isIdent && x.Name == "time" && t.Sel.Name == "Time" {
			return kindOrdered, true, true
		}
		return kindEquality, false, true // e.g. sql.NullString, datatypes.JSON
	case *ast.StarExpr:
		inner, innerOrderable, innerOK := classify(t.X, declared)
		if !innerOK || inner == kindPointer {
			return 0, false, false
		}
		return kindPointer, innerOrderable, true
	}
	return 0, false, false
}

func collectPackages(expr ast.Expr, used map[string]bool) {
	ast.Inspect(expr, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if x, ok := sel.X.(*ast.Ident); ok {
				used[x.Name] = true
			}
			return false
		}
		return true
	})
}

// resolveImports maps the package names used by field types to the source file's import lines.
func resolveImports(f *ast.File, used map[string]bool) ([]string, error) {
	var lines []string
	for name := range used {
		line := ""
		for _, spec := range f.Imports {
			p := strings.Trim(spec.Path.Value, `"`)
			if spec.Name != nil {
				if spec.Name.Name == name {
					line = spec.Name.Name + " " + spec.Path.Value
				}
				continue
			}
			if defaultName(p) == name {
				line = spec.Path.Value
			}
		}
		if line == "" {
			return nil, fmt.Errorf("cannot find the import for package %q; give the import an explicit name", name)
		}
		lines = append(lines, line)
	}
	sort.Strings(lines)
	return lines, nil
}

// defaultName is the conventional package name of an import path: its last element without a major-version suffix
// or a "go-" prefix.
func defaultName(importPath string) string {
	base := path.Base(importPath)
	if strings.HasPrefix(base, "v") && len(base) > 1 && strings.Trim(base[1:], "0123456789") == "" {
		base = path.Base(path.Dir(importPath))
	}
	base = strings.TrimPrefix(base, "go-")
	if i := strings.IndexByte(base, '.'); i > 0 {
		base = base[:i]
	}
	return strings.ReplaceAll(base, "-", "_")
}
