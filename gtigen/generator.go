// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gtigen

import (
	"bytes"
	"fmt"
	"go/ast"
	"log"
	"strings"
	"text/template"

	"goki.dev/gengo"
	"goki.dev/grease"
	"goki.dev/gti"
	"goki.dev/ordmap"
	"golang.org/x/tools/go/packages"
)

// Generator holds the state of the generator.
// It is primarily used to buffer the output.
type Generator struct {
	Config  *Config                            // The configuration information
	Buf     bytes.Buffer                       // The accumulated output.
	Pkgs    []*packages.Package                // The packages we are scanning.
	Pkg     *packages.Package                  // The packages we are currently on.
	Types   []*Type                            // The types
	Methods *ordmap.Map[string, []*gti.Method] // The methods, keyed by the the full package name of the type of the receiver
	Funcs   *ordmap.Map[string, *gti.Func]     // The functions
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
	// we need a manual import of gti and ordmap because they are
	// external, but goimports will handle everything else
	gengo.PrintHeader(&g.Buf, g.Pkg.Name, "goki.dev/gti", "goki.dev/ordmap")
}

// Find goes through all of the types, functions, variables,
// and constants in the package, finds those marked with gti:add,
// and adds them to [Generator.Types] and [Generator.Funcs]
func (g *Generator) Find() error {
	g.Types = []*Type{}
	g.Methods = &ordmap.Map[string, []*gti.Method]{}
	g.Funcs = &ordmap.Map[string, *gti.Func]{}
	err := gengo.Inspect(g.Pkg, g.Inspect)
	if err != nil {
		return fmt.Errorf("error while inspecting: %w", err)
	}
	return nil
}

// AllowedEnumTypes are the types that can be used for enums
// that are not bit flags (bit flags can only be int64s).
// It is stored as a map for quick and convenient access.
var AllowedEnumTypes = map[string]bool{"int": true, "int64": true, "int32": true, "int16": true, "int8": true, "uint": true, "uint64": true, "uint32": true, "uint16": true, "uint8": true}

// Inspect looks at the given AST node and adds it
// to [Generator.Types] if it is marked with an appropriate
// comment directive. It returns whether the AST inspector should
// continue, and an error if there is one. It should only
// be called in [ast.Inspect].
func (g *Generator) Inspect(n ast.Node) (bool, error) {
	switch v := n.(type) {
	case *ast.GenDecl:
		return g.InspectGenDecl(v)
	case *ast.FuncDecl:
		return g.InspectFuncDecl(v)
	}
	return true, nil
}

// InspectGenDecl is the implementation of [Generator.Inspect]
// for [ast.GenDecl] nodes.
func (g *Generator) InspectGenDecl(gd *ast.GenDecl) (bool, error) {
	if gd.Doc == nil {
		return true, nil
	}
	hasAdd := false
	cfg := &Config{}
	*cfg = *g.Config
	dirs, hasAdd, err := LoadFromComment(gd.Doc, cfg)
	if err != nil {
		return false, err
	}
	if !hasAdd { // we must be told to add or we will not add
		return true, nil
	}
	doc := strings.TrimSuffix(gd.Doc.Text(), "\n")
	for _, spec := range gd.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok {
			return true, nil
		}
		typ := &Type{
			Name:       ts.Name.Name,
			FullName:   g.Pkg.PkgPath + "." + ts.Name.Name,
			Type:       ts,
			Doc:        doc,
			Directives: dirs,
			Config:     cfg,
		}
		st, ok := ts.Type.(*ast.StructType)
		if ok {
			fields, err := GetFields(st.Fields, cfg)
			if err != nil {
				return false, err
			}
			typ.Fields = fields
		}
		g.Types = append(g.Types, typ)
	}
	return true, nil
}

// InspectFuncDecl is the implementation of [Generator.Inspect]
// for [ast.FuncDecl] nodes.
func (g *Generator) InspectFuncDecl(fd *ast.FuncDecl) (bool, error) {
	if fd.Doc == nil {
		return true, nil
	}
	cfg := &Config{}
	*cfg = *g.Config
	dirs, hasAdd, err := LoadFromComment(fd.Doc, cfg)
	if err != nil {
		return false, err
	}
	if !hasAdd { // we must be told to add or we will not add
		return true, nil
	}
	doc := strings.TrimSuffix(fd.Doc.Text(), "\n")

	if fd.Recv == nil {
		fun := &gti.Func{
			Name:       fd.Name.Name,
			Doc:        doc,
			Directives: dirs,
		}
		args, err := GetFields(fd.Type.Params, cfg)
		if err != nil {
			return false, fmt.Errorf("error getting function args: %w", err)
		}
		fun.Args = args
		rets, err := GetFields(fd.Type.Results, cfg)
		if err != nil {
			return false, fmt.Errorf("error getting function return values: %w", err)
		}
		fun.Returns = rets
		g.Funcs.Add(fun.Name, fun)
	} else {
		method := &gti.Method{
			Name:       fd.Name.Name,
			Doc:        doc,
			Directives: dirs,
		}
		args, err := GetFields(fd.Type.Params, cfg)
		if err != nil {
			return false, fmt.Errorf("error getting method args: %w", err)
		}
		method.Args = args
		rets, err := GetFields(fd.Type.Results, cfg)
		if err != nil {
			return false, fmt.Errorf("error getting method return values: %w", err)
		}
		method.Returns = rets

		typ := fd.Recv.List[0].Type
		typnm := fmt.Sprintf("%s.%v", g.Pkg.PkgPath, typ)
		g.Methods.Add(typnm, append(g.Methods.ValByKey(typnm), method))
	}

	return true, nil
}

// GetFields creates and returns a new [gti.Fields] object
// from the given [ast.FieldList], in the context of the
// given surrounding config. If the given field list is
// nil, GetFields still returns an empty but valid
// [gti.Fields] value and no error.
func GetFields(list *ast.FieldList, cfg *Config) (*gti.Fields, error) {
	res := &gti.Fields{}
	if list == nil {
		return res, nil
	}
	for _, field := range list.List {
		// if we have no name, fall back on type name
		name := fmt.Sprintf("%v", field.Type)
		if len(field.Names) > 0 {
			name = field.Names[0].Name
		}
		dirs := gti.Directives{}
		if field.Doc != nil {
			lcfg := &Config{}
			*lcfg = *cfg
			sdirs, _, err := LoadFromComment(field.Doc, lcfg)
			if err != nil {
				return nil, err
			}
			dirs = sdirs
		}
		fo := &gti.Field{
			Name:       name,
			Doc:        strings.TrimSuffix(field.Doc.Text(), "\n"),
			Directives: dirs,
		}
		res.Add(name, fo)
	}
	return res, nil
}

// LoadFromComment processes the given comment group, setting the
// values of the given config object based on any gti directives
// in the comment group, and returning all directives found, whether
// there was a gti:add directive, and any error.
func LoadFromComment(c *ast.CommentGroup, cfg *Config) (gti.Directives, bool, error) {
	dirs := gti.Directives{}
	hasAdd := false
	for _, c := range c.List {
		dir, err := grease.ParseDirective(c.Text)
		if err != nil {
			return nil, false, fmt.Errorf("error parsing comment directive from %q: %w", c.Text, err)
		}
		if dir == nil {
			continue
		}
		if dir.Tool == "gti" {
			if dir.Directive == "add" {
				hasAdd = true
				leftovers, err := grease.SetFromArgs(cfg, dir.Args)
				if err != nil {
					return nil, false, fmt.Errorf("error setting config info from comment directive args: %w (from directive %q)", err, c.Text)
				}
				if len(leftovers) > 0 {
					return nil, false, fmt.Errorf("expected 0 positional arguments but got %d (list: %v) (from directive %q)", len(leftovers), leftovers, c.Text)
				}
			} else {
				return nil, false, fmt.Errorf("unrecognized gti directive %q (from %q)", dir.Directive, c.Text)
			}
		}
		dirs = append(dirs, dir)
	}
	return dirs, hasAdd, nil
}

// Generate produces the code for the types
// stored in [Generator.Types] and stores them in
// [Generator.Buf]. It returns whether there were
// any types to generate methods for, and
// any error that occurred.
func (g *Generator) Generate() (bool, error) {
	if len(g.Types) == 0 {
		return false, nil
	}
	for _, typ := range g.Types {
		typ.Methods = &gti.Methods{}
		for _, meth := range g.Methods.ValByKey(typ.FullName) {
			typ.Methods.Add(meth.Name, meth)
		}
		g.ExecTmpl(TypeTmpl, typ)
	}
	for _, fun := range g.Funcs.Order {
		g.ExecTmpl(FuncTmpl, fun.Val)
	}
	return true, nil
}

// ExecTmpl executes the given template with the given data and
// writes the result to [Generator.Buf]. It fatally logs any error.
// All gtigen templates take a [*Type] or [*gti.Func] as their data.
func (g *Generator) ExecTmpl(t *template.Template, data any) {
	err := t.Execute(&g.Buf, data)
	if err != nil {
		log.Fatalf("programmer error: internal error: error executing template: %v", err)
	}
}

// Write formats the data in the the Generator's buffer
// ([Generator.Buf]) and writes it to the file specified by
// [Generator.Config.Output].
func (g *Generator) Write() error {
	return gengo.Write(gengo.Filepath(g.Pkg, g.Config.Output), g.Buf.Bytes(), nil)
}
