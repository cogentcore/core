// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gtigen

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/types"
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
	Config     *Config                               // The configuration information
	Buf        bytes.Buffer                          // The accumulated output.
	Pkgs       []*packages.Package                   // The packages we are scanning.
	Pkg        *packages.Package                     // The packages we are currently on.
	Types      []*Type                               // The types
	Methods    *ordmap.Map[string, []*gti.Method]    // The methods, keyed by the the full package name of the type of the receiver
	Funcs      *ordmap.Map[string, *gti.Func]        // The functions
	Interfaces *ordmap.Map[string, *types.Interface] // The cached interfaces, created from [Config.InterfaceConfigs]
}

// NewGenerator returns a new generator with the
// given configuration information and parsed packages.
func NewGenerator(config *Config, pkgs []*packages.Package) *Generator {
	return &Generator{Config: config, Pkgs: pkgs}
}

// PackageModes returns the package load modes needed for gtigen,
// based on the given config information.
func PackageModes(cfg *Config) packages.LoadMode {
	res := packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes | packages.NeedSyntax | packages.NeedTypesInfo
	// we only need deps if we are checking for interface impls
	if len(cfg.InterfaceConfigs) > 0 {
		res |= packages.NeedDeps
	}
	return res
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
	if len(g.Config.InterfaceConfigs) > 0 {
		g.Interfaces = &ordmap.Map[string, *types.Interface]{}
		err := g.GetInterfaces([]*types.Package{g.Pkg.Types})
		if err != nil {
			return fmt.Errorf("error getting interface objects from interface configs: %w", err)
		}
	}
	g.Types = []*Type{}
	g.Methods = &ordmap.Map[string, []*gti.Method]{}
	g.Funcs = &ordmap.Map[string, *gti.Func]{}
	err := gengo.Inspect(g.Pkg, g.Inspect)
	if err != nil {
		return fmt.Errorf("error while inspecting: %w", err)
	}
	return nil
}

// GetInterfaces sets [Generator.Interfaces] based on
// [Generator.Config.InterfaceConfigs], looking in the
// given packages. It is a recursive function that should
// not typically be called by end-user code.
func (g *Generator) GetInterfaces(pkgs []*types.Package) error {
	rpkgs := []*types.Package{}
	for _, pkg := range pkgs {
		for in := range g.Config.InterfaceConfigs {
			strs := strings.Split(in, ".")
			if len(strs) < 2 {
				return errors.New("expected something before and after dot in fully-qualified type name")
			}
			pkgnm := strs[len(strs)-2]
			if pkg.Name() == pkgnm {
				typnm := strs[len(strs)-1]
				typ := pkg.Scope().Lookup(typnm)
				if typ == nil {
					return fmt.Errorf("programmer error: internal error: could not find type %q in package %q (from interface config %q)", typnm, pkgnm, in)
				}
				tn, ok := typ.Type().(*types.Named)
				if !ok {
					return fmt.Errorf("programmer error: internal error: type %q is not a *types.Named but a %T (type value %v)", in, typ.Type(), typ.Type())
				}
				tint, ok := tn.Underlying().(*types.Interface)
				if !ok {
					return fmt.Errorf("programmer error: internal error: underlying type of type %q is not a *types.Interface but a %T (type value %v)", in, tn.Underlying(), tn.Underlying())
				}
				g.Interfaces.Add(in, tint)
			}
		}
		rpkgs = append(rpkgs, pkg.Imports()...)
	}
	if len(pkgs) > 0 {
		return g.GetInterfaces(rpkgs)
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
	hasAdd := false
	cfg := &Config{}
	*cfg = *g.Config
	dirs, hasAdd, err := LoadFromComment(gd.Doc, cfg)
	if err != nil {
		return false, err
	}
	if !hasAdd && !cfg.AddTypes && len(cfg.InterfaceConfigs) == 0 { // we must be told to add or we will not add, and if we have interface configs we will handle adding later
		return true, nil
	}
	doc := strings.TrimSuffix(gd.Doc.Text(), "\n")
	for _, spec := range gd.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok {
			return true, nil
		}
		if len(cfg.InterfaceConfigs) > 0 {
			hasInt := false
			typ := g.Pkg.TypesInfo.Defs[ts.Name].Type()
			for in, ic := range cfg.InterfaceConfigs {
				iface := g.Interfaces.ValByKey(in)
				if iface == nil {
					return false, fmt.Errorf("programmer error: internal error: missing interface object for interface %q", in)
				}
				if !types.Implements(typ, iface) {
					continue
				}
				*cfg = *ic
				dirs, hasAdd, err = LoadFromComment(gd.Doc, cfg)
				hasInt = true
				if err != nil {
					return false, err
				}
				if !hasAdd && !cfg.AddTypes { // we must be told to add or we will not add
					return true, nil
				}
			}
			if !hasInt && !hasAdd && !cfg.AddTypes { // we must be told to add or we will not add
				return true, nil
			}
		}
		typ := &Type{
			Name:       ts.Name.Name,
			FullName:   FullName(g.Pkg, ts.Name.Name),
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
	cfg := &Config{}
	*cfg = *g.Config
	dirs, hasAdd, err := LoadFromComment(fd.Doc, cfg)
	if err != nil {
		return false, err
	}
	doc := strings.TrimSuffix(fd.Doc.Text(), "\n")

	if fd.Recv == nil {
		if !hasAdd && !cfg.AddFuncs { // we must be told to add or we will not add
			return true, nil
		}
		fun := &gti.Func{
			Name:       FullName(g.Pkg, fd.Name.Name),
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
		if !hasAdd && !cfg.AddMethods { // we must be told to add or we will not add
			return true, nil
		}
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
		typnm := FullName(g.Pkg, fmt.Sprintf("%v", typ))
		g.Methods.Add(typnm, append(g.Methods.ValByKey(typnm), method))
	}

	return true, nil
}

// FullName returns the fully qualified name of an identifier
// in the given package with the given name.
func FullName(pkg *packages.Package, name string) string {
	// idents in main packages are just "main.IdentName"
	if pkg.Name == "main" {
		return "main." + name
	}
	return pkg.PkgPath + "." + name
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
// there was a gti:add directive, and any error. If the given
// documentation is nil, LoadFromComment still returns an empty but valid
// [gti.Directives] value, false, and no error.
func LoadFromComment(c *ast.CommentGroup, cfg *Config) (gti.Directives, bool, error) {
	dirs := gti.Directives{}
	hasAdd := false
	if c == nil {
		return dirs, false, nil
	}
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
	if len(g.Types) == 0 && g.Funcs.Len() == 0 {
		return false, nil
	}
	for _, typ := range g.Types {
		typ.Methods = &gti.Methods{}
		for _, meth := range g.Methods.ValByKey(typ.FullName) {
			typ.Methods.Add(meth.Name, meth)
		}
		g.ExecTmpl(TypeTmpl, typ)
		for _, tmpl := range typ.Config.Templates {
			g.ExecTmpl(tmpl, typ)
		}
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
