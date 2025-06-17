// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on http://github.com/dmarkham/enumer and
// golang.org/x/tools/cmd/stringer:

// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package enumgen

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"html"
	"log/slog"
	"os"
	"strings"
	"text/template"

	"cogentcore.org/core/base/generate"
	"cogentcore.org/core/cli"
	"golang.org/x/tools/go/packages"
)

// Generator holds the state of the generator.
// It is primarily used to buffer the output.
type Generator struct {
	Config *Config             // The configuration information
	Buf    bytes.Buffer        // The accumulated output.
	Pkgs   []*packages.Package // The packages we are scanning.
	Pkg    *packages.Package   // The packages we are currently on.
	Types  []*Type             // The enum types
}

// NewGenerator returns a new generator with the
// given configuration information and parsed packages.
func NewGenerator(config *Config, pkgs []*packages.Package) *Generator {
	return &Generator{Config: config, Pkgs: pkgs}
}

// PackageModes returns the package load modes needed for this generator
func PackageModes() packages.LoadMode {
	return packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes | packages.NeedSyntax | packages.NeedTypesInfo
}

// Printf prints the formatted string to the
// accumulated output in [Generator.Buf]
func (g *Generator) Printf(format string, args ...any) {
	fmt.Fprintf(&g.Buf, format, args...)
}

// PrintHeader prints the header and package clause
// to the accumulated output
func (g *Generator) PrintHeader() {
	// we need a manual import of enums because it is
	// external, but goimports will handle everything else
	generate.PrintHeader(&g.Buf, g.Pkg.Name, "cogentcore.org/core/enums")
}

// FindEnumTypes goes through all of the types in the package
// and finds all integer (signed or unsigned) types labeled with enums:enum
// or enums:bitflag. It stores the resulting types in [Generator.Types].
func (g *Generator) FindEnumTypes() error {
	g.Types = []*Type{}
	return generate.Inspect(g.Pkg, g.InspectForType, "enumgen.go", "typegen.go")
}

// AllowedEnumTypes are the types that can be used for enums
// that are not bit flags (bit flags can only be int64s).
// It is stored as a map for quick and convenient access.
var AllowedEnumTypes = map[string]bool{"int": true, "int64": true, "int32": true, "int16": true, "int8": true, "uint": true, "uint64": true, "uint32": true, "uint16": true, "uint8": true}

// InspectForType looks at the given AST node and adds it
// to [Generator.Types] if it is marked with an appropriate
// comment directive. It returns whether the AST inspector should
// continue, and an error if there is one. It should only
// be called in [ast.Inspect].
func (g *Generator) InspectForType(n ast.Node) (bool, error) {
	ts, ok := n.(*ast.TypeSpec)
	if !ok {
		return true, nil
	}
	if ts.Comment == nil {
		return true, nil
	}
	for _, c := range ts.Comment.List {
		dir, err := cli.ParseDirective(c.Text)
		if err != nil {
			return false, fmt.Errorf("error parsing comment directive %q: %w", c.Text, err)
		}
		if dir == nil {
			continue
		}
		if dir.Tool != "enums" {
			continue
		}
		if dir.Directive != "enum" && dir.Directive != "bitflag" {
			return false, fmt.Errorf("unrecognized enums directive %q (from %q)", dir.Directive, c.Text)
		}

		typnm := types.ExprString(ts.Type)
		// ident, ok := ts.Type.(*ast.Ident)
		// if !ok {
		// 	return false, fmt.Errorf("type of enum type (%v) is %T, not *ast.Ident (try using a standard [un]signed integer type instead)", ts.Type, ts.Type)
		// }
		cfg := &Config{}
		*cfg = *g.Config
		leftovers, err := cli.SetFromArgs(cfg, dir.Args, cli.ErrNotFound)
		if err != nil {
			return false, fmt.Errorf("error setting config info from comment directive args: %w (from directive %q)", err, c.Text)
		}
		if len(leftovers) > 0 {
			return false, fmt.Errorf("expected 0 positional arguments but got %d (list: %v) (from directive %q)", len(leftovers), leftovers, c.Text)
		}

		typ := g.Pkg.TypesInfo.Defs[ts.Name].Type()
		utyp := typ.Underlying()

		tt := &Type{Name: ts.Name.Name, Type: ts, Config: cfg}
		// if our direct type isn't the same as our underlying type, we are extending our direct type
		if cfg.Extend && typnm != utyp.String() {
			tt.Extends = typnm
		}
		switch dir.Directive {
		case "enum":
			if !AllowedEnumTypes[utyp.String()] {
				return false, fmt.Errorf("enum type %s is not allowed; try using a standard [un]signed integer type instead", typnm)
			}
			tt.IsBitFlag = false
		case "bitflag":
			if utyp.String() != "int64" {
				return false, fmt.Errorf("bit flag enum type %s is not allowed; bit flag enums must be of type int64", typnm)
			}
			tt.IsBitFlag = true
		}
		g.Types = append(g.Types, tt)

	}
	return true, nil
}

// Generate produces the enum methods for the types
// stored in [Generator.Types] and stores them in
// [Generator.Buf]. It returns whether there were
// any enum types to generate methods for, and
// any error that occurred.
func (g *Generator) Generate() (bool, error) {
	if len(g.Types) == 0 {
		return false, nil
	}
	for _, typ := range g.Types {
		values := make([]Value, 0, 100)
		for _, file := range g.Pkg.Syntax {
			if generate.ExcludeFile(g.Pkg, file, "enumgen.go", "typegen.go") {
				continue
			}
			var terr error
			ast.Inspect(file, func(n ast.Node) bool {
				if terr != nil {
					return false
				}
				vals, cont, err := g.GenDecl(n, file, typ)
				if err != nil {
					terr = err
				} else {
					values = append(values, vals...)
				}
				return cont
			})
			if terr != nil {
				return true, fmt.Errorf("Generate: error parsing declaration clauses: %w", terr)
			}
		}

		if len(values) == 0 {
			return true, errors.New("no values defined for type " + typ.Name)
		}

		g.TrimValueNames(values, typ.Config)

		err := g.TransformValueNames(values, typ.Config)
		if err != nil {
			return true, fmt.Errorf("error transforming value names: %w", err)
		}

		g.PrefixValueNames(values, typ.Config)

		values = SortValues(values, typ)

		g.BuildBasicMethods(values, typ)
		if typ.IsBitFlag {
			g.BuildBitFlagMethods(values, typ)
		}

		if typ.Config.Text {
			g.BuildTextMethods(values, typ)
		}
		if typ.Config.SQL {
			g.AddValueAndScanMethod(typ)
		}
		if typ.Config.GQL {
			g.BuildGQLMethods(values, typ)
		}
	}
	return true, nil
}

// GenDecl processes one declaration clause.
// It returns whether the AST inspector should continue,
// and an error if there is one. It should only be
// called in [ast.Inspect].
func (g *Generator) GenDecl(node ast.Node, file *ast.File, typ *Type) ([]Value, bool, error) {
	decl, ok := node.(*ast.GenDecl)
	if !ok || decl.Tok != token.CONST {
		// We only care about const declarations.
		return nil, true, nil
	}
	vals := []Value{}
	// The name of the type of the constants we are declaring.
	// Can change if this is a multi-element declaration.
	typName := ""
	// Loop over the elements of the declaration. Each element is a ValueSpec:
	// a list of names possibly followed by a type, possibly followed by values.
	// If the type and value are both missing, we carry down the type (and value,
	// but the "go/types" package takes care of that).
	for _, spec := range decl.Specs {
		vspec := spec.(*ast.ValueSpec) // Guaranteed to succeed as this is CONST.
		if vspec.Type == nil && len(vspec.Values) > 0 {
			// "X = 1". With no type but a value, the constant is untyped.
			// Skip this vspec and reset the remembered type.
			typName = ""
			continue
		}
		if vspec.Type != nil {
			// "X T". We have a type. Remember it.
			ident, ok := vspec.Type.(*ast.Ident)
			if !ok {
				continue
			}
			typName = ident.Name
		}
		if typName != typ.Name {
			// This is not the type we're looking for.
			continue
		}
		// We now have a list of names (from one line of source code) all being
		// declared with the desired type.
		// Grab their names and actual values and store them in f.values.
		for _, n := range vspec.Names {
			if n.Name == "_" {
				continue
			}
			// This dance lets the type checker find the values for us. It's a
			// bit tricky: look up the object declared by the n, find its
			// types.Const, and extract its value.
			obj, ok := g.Pkg.TypesInfo.Defs[n]
			if !ok {
				return nil, false, errors.New("no value for constant " + n.String())
			}
			info := obj.Type().Underlying().(*types.Basic).Info()
			if info&types.IsInteger == 0 {
				return nil, false, errors.New("can't handle non-integer constant type " + typName)
			}
			value := obj.(*types.Const).Val() // Guaranteed to succeed as this is CONST.
			if value.Kind() != constant.Int {
				return nil, false, errors.New("can't happen: constant is not an integer " + n.String())
			}
			i64, isInt := constant.Int64Val(value)
			u64, isUint := constant.Uint64Val(value)
			if !isInt && !isUint {
				return nil, false, errors.New("internal error: value of " + n.String() + " is not an integer: " + value.String())
			}
			if !isUint {
				i64 = int64(u64)
			}
			v := Value{
				OriginalName: n.Name,
				Name:         n.Name,
				Desc:         html.EscapeString(strings.Join(strings.Fields(vspec.Doc.Text()), " ")), // need to collapse whitespace and escape
				Value:        i64,
				Signed:       info&types.IsUnsigned == 0,
				Str:          value.String(),
			}
			if c := vspec.Comment; typ.Config.LineComment && c != nil && len(c.List) == 1 {
				v.Name = strings.TrimSpace(c.Text())
			}

			vals = append(vals, v)
		}
	}
	return vals, false, nil
}

// ExecTmpl executes the given template with the given type and
// writes the result to [Generator.Buf]. It fatally logs any error.
// All enumgen templates take a [Type] as their data.
func (g *Generator) ExecTmpl(t *template.Template, typ *Type) {
	err := t.Execute(&g.Buf, typ)
	if err != nil {
		slog.Error("programmer error: internal error: error executing template", "err", err)
		os.Exit(1)
	}
}

// Write formats the data in the the Generator's buffer
// ([Generator.Buf]) and writes it to the file specified by
// [Generator.Config.Output].
func (g *Generator) Write() error {
	return generate.Write(generate.Filepath(g.Pkg, g.Config.Output), g.Buf.Bytes(), nil)
}
