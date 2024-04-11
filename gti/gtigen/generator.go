// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gtigen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/types"
	"maps"
	"os"
	"slices"
	"strings"
	"text/template"

	"log/slog"

	"cogentcore.org/core/cli"
	"cogentcore.org/core/generate"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/ordmap"
	"cogentcore.org/core/strcase"
	"golang.org/x/tools/go/packages"
)

// Generator holds the state of the generator.
// It is primarily used to buffer the output.
type Generator struct {
	Config     *Config                              // The configuration information
	Buf        bytes.Buffer                         // The accumulated output.
	Pkgs       []*packages.Package                  // The packages we are scanning.
	Pkg        *packages.Package                    // The packages we are currently on.
	File       *ast.File                            // The file we are currently on.
	Cmap       ast.CommentMap                       // The comment map for the file we are currently on.
	Types      []*Type                              // The types
	Methods    ordmap.Map[string, []gti.Method]     // The methods, keyed by the the full package name of the type of the receiver
	Funcs      ordmap.Map[string, gti.Func]         // The functions
	Interfaces ordmap.Map[string, *types.Interface] // The cached interfaces, created from [Config.InterfaceConfigs]
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
	if cfg.InterfaceConfigs.Len() > 0 {
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
	generate.PrintHeader(&g.Buf, g.Pkg.Name, "cogentcore.org/core/gti", "cogentcore.org/core/ordmap")
}

// Find goes through all of the types, functions, variables,
// and constants in the package, finds those marked with gti:add,
// and adds them to [Generator.Types] and [Generator.Funcs]
func (g *Generator) Find() error {
	err := g.GetInterfaces()
	if err != nil {
		return err
	}
	g.Types = []*Type{}
	err = generate.Inspect(g.Pkg, g.Inspect)
	if err != nil {
		return fmt.Errorf("error while inspecting: %w", err)
	}
	return nil
}

// GetInterfaces sets [Generator.Interfaces] based on
// [Generator.Config.InterfaceConfigs]. It should typically not
// be called by end-user code.
func (g *Generator) GetInterfaces() error {
	if g.Config.InterfaceConfigs.Len() == 0 {
		return nil
	}
	for _, typ := range g.Pkg.TypesInfo.Types {
		nm := typ.Type.String()
		if _, ok := g.Config.InterfaceConfigs.ValueByKeyTry(nm); ok {
			utyp := typ.Type.Underlying()
			iface, ok := utyp.(*types.Interface)
			if !ok {
				return fmt.Errorf("invalid InterfaceConfigs value: type %q is not a *types.Interface but a %T (type value %v)", nm, utyp, utyp)
			}
			g.Interfaces.Add(nm, iface)
		}
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
	case *ast.File:
		g.File = v
		g.Cmap = ast.NewCommentMap(g.Pkg.Fset, v, v.Comments)
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
	doc := strings.TrimSuffix(gd.Doc.Text(), "\n")
	for _, spec := range gd.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok {
			return true, nil
		}
		cfg := &Config{}
		*cfg = *g.Config

		if cfg.InterfaceConfigs.Len() > 0 {
			typ := g.Pkg.TypesInfo.Defs[ts.Name].Type()
			if !types.IsInterface(typ) {
				for _, kv := range cfg.InterfaceConfigs.Order {
					in := kv.Key
					ic := kv.Value
					iface := g.Interfaces.ValueByKey(in)
					if iface == nil {
						slog.Info("missing interface object", "interface", in)
						continue
					}
					if !types.Implements(typ, iface) && !types.Implements(types.NewPointer(typ), iface) { // either base type or pointer can implement
						continue
					}
					*cfg = *ic
				}
			}
		}

		// By default, we use the comments on the GenDecl, as that
		// is where they are normally stored. However, when there
		// are comments on the type spec itself, that means we are
		// probably in a type block, and thus we must use the comments
		// on the type spec itself.
		commentNode := ast.Node(gd)
		if ts.Doc != nil || ts.Comment != nil {
			commentNode = ts
		}
		if ts.Doc != nil {
			doc = strings.TrimSuffix(ts.Doc.Text(), "\n")
		}

		dirs, hasAdd, hasSkip, err := g.LoadFromNodeComments(cfg, commentNode)
		if err != nil {
			return false, err
		}
		if (!hasAdd && !cfg.AddTypes) || hasSkip { // we must be told to add or we will not add
			return true, nil
		}

		typ := &Type{
			Type: gti.Type{
				Name:       FullName(g.Pkg, ts.Name.Name),
				IDName:     strcase.ToKebab(ts.Name.Name),
				Doc:        doc,
				Directives: dirs,
			},
			LocalName: ts.Name.Name,
			AST:       ts,
			Pkg:       g.Pkg.Name,
			Config:    cfg,
		}
		st, ok := ts.Type.(*ast.StructType)
		if ok && st.Fields != nil {
			emblist := &ast.FieldList{}
			delOff := 0
			for i := range len(st.Fields.List) {
				i -= delOff
				field := st.Fields.List[i]
				// if we have no names, we are embed, so add to embeds and remove from fields
				if len(field.Names) == 0 {
					emblist.List = append(emblist.List, field)
					st.Fields.List = slices.Delete(st.Fields.List, i, i+1)
					delOff++
				}
			}

			embeds, err := g.GetFields(emblist, cfg)
			if err != nil {
				return false, err
			}
			typ.Embeds = embeds

			fields, err := g.GetFields(st.Fields, cfg)
			if err != nil {
				return false, err
			}
			typ.Fields = fields

			typ.EmbeddedFields = Fields{LocalTypes: map[string]string{}, Tags: map[string]string{}}
			tp := g.Pkg.TypesInfo.TypeOf(ts.Type)
			g.GetEmbeddedFields(&typ.EmbeddedFields, tp, tp)
		}
		g.Types = append(g.Types, typ)
	}
	return true, nil
}

// LocalTypeNameQualifier returns a [types.Qualifier] similar to that
// returned by [types.RelativeTo], but using the package name instead
// of the package path so that it can be used in code.
func LocalTypeNameQualifier(pkg *types.Package) types.Qualifier {
	if pkg == nil {
		return nil
	}
	return func(other *types.Package) string {
		if pkg == other {
			return "" // same package; unqualified
		}
		return other.Name()
	}
}

// GetEmbeddedFields recursively adds to the given set of embedded fields all of the embedded
// fields for the given type. It does not add the fields in the given starting type,
// as those fields aren't embedded.
func (g *Generator) GetEmbeddedFields(efields *Fields, typ, startTyp types.Type) {
	s, ok := typ.Underlying().(*types.Struct)
	if !ok {
		return
	}
	nf := s.NumFields()
	for i := 0; i < nf; i++ {
		f := s.Field(i)
		if f.Embedded() {
			g.GetEmbeddedFields(efields, f.Type(), startTyp)
			continue
		}
		// we don't add for start type, as field in that aren't embedded
		if typ == startTyp {
			continue
		}
		field := gti.Field{
			Name: f.Name(),
		}
		efields.Fields = append(efields.Fields, field)
		efields.LocalTypes[field.Name] = types.TypeString(f.Type(), LocalTypeNameQualifier(g.Pkg.Types))
		efields.Tags[field.Name] = s.Tag(i)
	}
}

// InspectFuncDecl is the implementation of [Generator.Inspect]
// for [ast.FuncDecl] nodes.
func (g *Generator) InspectFuncDecl(fd *ast.FuncDecl) (bool, error) {
	cfg := &Config{}
	*cfg = *g.Config
	dirs, hasAdd, hasSkip, err := g.LoadFromNodeComments(cfg, fd)
	if err != nil {
		return false, err
	}
	doc := strings.TrimSuffix(fd.Doc.Text(), "\n")

	if fd.Recv == nil {
		if (!hasAdd && !cfg.AddFuncs) || hasSkip { // we must be told to add or we will not add
			return true, nil
		}
		fun := gti.Func{
			Name:       FullName(g.Pkg, fd.Name.Name),
			Doc:        doc,
			Directives: dirs,
		}
		args, err := g.GetFields(fd.Type.Params, cfg)
		if err != nil {
			return false, fmt.Errorf("error getting function args: %w", err)
		}
		for _, arg := range args.Fields {
			fun.Args = append(fun.Args, arg.Name)
		}
		rets, err := g.GetFields(fd.Type.Results, cfg)
		if err != nil {
			return false, fmt.Errorf("error getting function return values: %w", err)
		}
		for _, ret := range rets.Fields {
			fun.Returns = append(fun.Returns, ret.Name)
		}
		g.Funcs.Add(fun.Name, fun)
	} else {
		if (!hasAdd && !cfg.AddMethods) || hasSkip { // we must be told to add or we will not add
			return true, nil
		}
		method := gti.Method{
			Name:       fd.Name.Name,
			Doc:        doc,
			Directives: dirs,
		}
		args, err := g.GetFields(fd.Type.Params, cfg)
		if err != nil {
			return false, fmt.Errorf("error getting method args: %w", err)
		}
		for _, arg := range args.Fields {
			method.Args = append(method.Args, arg.Name)
		}
		rets, err := g.GetFields(fd.Type.Results, cfg)
		if err != nil {
			return false, fmt.Errorf("error getting method return values: %w", err)
		}
		for _, ret := range rets.Fields {
			method.Returns = append(method.Returns, ret.Name)
		}

		typ := fd.Recv.List[0].Type
		// get rid of any pointer receiver
		tnm := strings.TrimPrefix(types.ExprString(typ), "*")
		typnm := FullName(g.Pkg, tnm)
		g.Methods.Add(typnm, append(g.Methods.ValueByKey(typnm), method))
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
func (g *Generator) GetFields(list *ast.FieldList, cfg *Config) (Fields, error) {
	res := Fields{LocalTypes: map[string]string{}, Tags: map[string]string{}}
	if list == nil {
		return res, nil
	}
	for _, field := range list.List {
		ltn := types.ExprString(field.Type)
		ftyp := g.Pkg.TypesInfo.TypeOf(field.Type)
		tn := ftyp.String()

		switch ftyp.(type) {
		case *types.Slice, *types.Array, *types.Map:
		default:
			// if the type is not a slice, array, or map, we get the name of the type
			// before anything involving square brackets so that generic types don't confuse it
			tn, _, _ = strings.Cut(tn, "[")
			tn, _, _ = strings.Cut(tn, "]")
		}

		name := ""
		if len(field.Names) == 1 {
			name = field.Names[0].Name
		} else if len(field.Names) == 0 {
			// if we have no name, fall back on type name
			name = tn
			// we must get rid of any package name, as field
			// names never have package names
			li := strings.LastIndex(name, ".")
			if li >= 0 {
				name = name[li+1:] // need to get rid of .
			}
		} else {
			// if we have more than one name, that typically indicates
			// type-omitted arguments (eg: "func(x, y float32)"), so
			// we handle all of the names seperately here and then continue.
			for _, nm := range field.Names {
				nfield := *field
				nfield.Names = []*ast.Ident{nm}
				nlist := &ast.FieldList{List: []*ast.Field{&nfield}}
				nfields, err := g.GetFields(nlist, cfg)
				if err != nil {
					return res, err
				}
				res.Fields = append(res.Fields, nfields.Fields...)
				maps.Copy(res.LocalTypes, nfields.LocalTypes)
				maps.Copy(res.Tags, nfields.Tags)
			}
			continue
		}
		fo := gti.Field{
			Name: name,
			Doc:  strings.TrimSuffix(field.Doc.Text(), "\n"),
		}
		res.Fields = append(res.Fields, fo)

		res.LocalTypes[name] = ltn

		tag := ""
		if field.Tag != nil {
			// need to get rid of leading and trailing backquotes
			tag = strings.TrimPrefix(strings.TrimSuffix(field.Tag.Value, "`"), "`")
		}
		res.Tags[name] = tag
	}
	return res, nil
}

// LoadFromNodeComments is a helper function that calls [LoadFromComments] with the correctly
// filtered comment map comments of the given node.
func (g *Generator) LoadFromNodeComments(cfg *Config, n ast.Node) (dirs []gti.Directive, hasAdd bool, hasSkip bool, err error) {
	cs := g.Cmap.Filter(n).Comments()
	tf := g.Pkg.Fset.File(g.File.FileStart)
	np := tf.Line(n.Pos())
	keep := []*ast.CommentGroup{}
	for _, c := range cs {
		// if the comment's line is after ours, we ignore it, as it is likely associated with something else
		if tf.Line(c.Pos()) > np {
			continue
		}
		keep = append(keep, c)
	}
	return LoadFromComments(cfg, keep...)
}

// LoadFromComments is a helper function that combines the results of [LoadFromComment]
// for the given comment groups.
func LoadFromComments(cfg *Config, c ...*ast.CommentGroup) (dirs []gti.Directive, hasAdd bool, hasSkip bool, err error) {
	for _, cg := range c {
		cdirs, cadd, cskip, err := LoadFromComment(cg, cfg)
		if err != nil {
			return nil, false, false, err
		}
		dirs = append(dirs, cdirs...)
		hasAdd = hasAdd || cadd
		hasSkip = hasSkip || cskip
	}
	return
}

// LoadFromComment processes the given comment group, setting the
// values of the given config object based on any gti directives
// in the comment group, and returning all directives found, whether
// there was a gti:add directive, and any error. If the given
// documentation is nil, LoadFromComment still returns an empty but valid
// [gti.Directives] value, false, and no error.
func LoadFromComment(c *ast.CommentGroup, cfg *Config) (dirs []gti.Directive, hasAdd bool, hasSkip bool, err error) {
	if c == nil {
		return
	}
	for _, c := range c.List {
		dir, err := cli.ParseDirective(c.Text)
		if err != nil {
			return nil, false, false, fmt.Errorf("error parsing comment directive from %q: %w", c.Text, err)
		}
		if dir == nil {
			continue
		}
		if dir.Tool == "gti" && dir.Directive == "add" {
			hasAdd = true
		}
		if dir.Tool == "gti" {
			if dir.Directive == "skip" {
				hasSkip = true
			}
			if dir.Directive == "add" || dir.Directive == "skip" {
				leftovers, err := cli.SetFromArgs(cfg, dir.Args, cli.ErrNotFound)
				if err != nil {
					return nil, false, false, fmt.Errorf("error setting config info from comment directive args: %w (from directive %q)", err, c.Text)
				}
				if len(leftovers) > 0 {
					return nil, false, false, fmt.Errorf("expected 0 positional arguments but got %d (list: %v) (from directive %q)", len(leftovers), leftovers, c.Text)
				}
			} else {
				return nil, false, false, fmt.Errorf("unrecognized gti directive %q (from %q)", dir.Directive, c.Text)
			}
		}
		dirs = append(dirs, *dir)
	}
	return dirs, hasAdd, hasSkip, nil
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
		typ.Methods = append(typ.Methods, g.Methods.ValueByKey(typ.Name)...)
		g.ExecTmpl(TypeTmpl, typ)
		for _, tmpl := range typ.Config.Templates {
			g.ExecTmpl(tmpl, typ)
		}
		if typ.Config.Setters {
			g.ExecTmpl(SetterMethodsTmpl, typ)
		}
	}
	for _, fun := range g.Funcs.Order {
		g.ExecTmpl(FuncTmpl, fun.Value)
	}
	return true, nil
}

// ExecTmpl executes the given template with the given data and
// writes the result to [Generator.Buf]. It fatally logs any error.
// All gtigen templates take a [*Type] or [*gti.Func] as their data.
func (g *Generator) ExecTmpl(t *template.Template, data any) {
	err := t.Execute(&g.Buf, data)
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
