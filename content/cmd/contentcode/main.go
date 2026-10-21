// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command contentcode compiles the runnable Go code blocks (```Go) of the
// markdown files of a content app into the app: it writes a Go file to the
// app's package that defines a function for each code block and registers it
// in [cogentcore.org/core/content.CompiledCode]. Content then runs those code
// blocks directly, without an interpreter such as yaegi: natively as Go, and
// in `core build js` apps as the GoScript output of the same code.
//
// Code blocks are written as yaegi runs them: a list of statements that
// use the parent widget b and the packages of the yaegicore symbols (core,
// styles, ...) without importing them, or a program with a main function.
// Code blocks that do not type check are left out (and listed in the output
// file); an unused variable is used with `_ = x` instead, since yaegi allows
// it.
//
// Usage, typically from a go:generate directive in the app's main package:
//
//	contentcode [-content content] [-o contentcode_gen.go] [-import name=path]...
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// defaultImports are the packages that code blocks can use without importing
// them: those of yaegicore/basesymbols and yaegicore/coresymbols.
var defaultImports = []string{
	"cogentcore.org/core/base/errors",
	"cogentcore.org/core/base/fileinfo",
	"cogentcore.org/core/base/fsx",
	"cogentcore.org/core/base/iox/imagex",
	"cogentcore.org/core/base/labels",
	"cogentcore.org/core/base/num",
	"cogentcore.org/core/base/reflectx",
	"cogentcore.org/core/base/strcase",
	"cogentcore.org/core/colors",
	"cogentcore.org/core/colors/gradient",
	"cogentcore.org/core/content",
	"cogentcore.org/core/core",
	"cogentcore.org/core/cursors",
	"cogentcore.org/core/events",
	"cogentcore.org/core/filetree",
	"cogentcore.org/core/htmlcore",
	"cogentcore.org/core/icons",
	"cogentcore.org/core/keymap",
	"cogentcore.org/core/math32",
	"cogentcore.org/core/paint",
	"cogentcore.org/core/styles/abilities",
	"cogentcore.org/core/styles/states",
	"cogentcore.org/core/styles",
	"cogentcore.org/core/styles/units",
	"cogentcore.org/core/text/lines",
	"cogentcore.org/core/text/rich",
	"cogentcore.org/core/text/text",
	"cogentcore.org/core/text/textcore",
	"cogentcore.org/core/tree",
	"fmt",
	"image/color",
	"image/draw",
	"image",
	"log/slog",
	"math",
	"path/filepath",
	"reflect",
	"strconv",
	"strings",
	"time",
}

// generated names; the prefix keeps them apart from the app's names.
const (
	prefix    = "contentCode"
	parentVar = prefix + "Parent"
	outVar    = prefix + "Out"
)

type importFlags []string

func (f *importFlags) String() string     { return strings.Join(*f, ",") }
func (f *importFlags) Set(s string) error { *f = append(*f, s); return nil }

func main() {
	contentDir := flag.String("content", "content", "directory of the markdown content")
	output := flag.String("o", "contentcode_gen.go", "output Go file, in the current package")
	tags := flag.String("tags", "", "build tags for type checking")
	var imports importFlags
	flag.Var(&imports, "import", "extra package that code can use: path or name=path (repeatable)")
	flag.Parse()
	if err := run(*contentDir, *output, *tags, imports); err != nil {
		fmt.Fprintln(os.Stderr, "contentcode:", err)
		os.Exit(1)
	}
}

// block is one code block, and the Go code generated for it.
type block struct {
	file  string // markdown file
	line  int    // line of the code in the file
	src   string // source of the code block
	key   string // CodeKey of src
	name  string // name of the generated function
	code  string // generated Go declarations
	pkgs  map[string]bool
	err   string // why the block was left out
	start int    // first line of code in the output file
	end   int    // last line of code in the output file
}

func run(contentDir, output, tags string, extra []string) error {
	importPaths := map[string]string{} // package name -> path
	for _, p := range defaultImports {
		importPaths[pathName(p)] = p
	}
	for _, e := range extra {
		name, path, ok := strings.Cut(e, "=")
		if !ok {
			name, path = pathName(e), e
		}
		importPaths[name] = path
	}

	pkgName, err := packageName(tags)
	if err != nil {
		return err
	}
	blocks, err := readBlocks(contentDir)
	if err != nil {
		return err
	}
	for i, b := range blocks {
		b.name = fmt.Sprintf("%s%d", prefix, i)
		if err := b.generate(importPaths); err != nil {
			b.err = err.Error()
		}
	}

	abs, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	// type check and fix or leave out blocks until the file type checks
	var src []byte
	for iter := 0; ; iter++ {
		src = assemble(pkgName, blocks, importPaths)
		if iter == 20 {
			return fmt.Errorf("type checking did not converge")
		}
		errs, err := typeCheck(abs, src, tags)
		if err != nil {
			return err
		}
		if len(errs) == 0 {
			break
		}
		changed := false
		unused := map[*block]map[int][]string{}
		for _, e := range errs {
			b := blockAt(blocks, e.line)
			if b == nil {
				return fmt.Errorf("error outside of the code blocks: %s", e)
			}
			if name, ok := strings.CutPrefix(e.msg, "declared and not used: "); ok {
				if unused[b] == nil {
					unused[b] = map[int][]string{}
				}
				unused[b][e.line-b.start] = append(unused[b][e.line-b.start], name)
				continue
			}
			if b.err == "" {
				b.err = e.msg
				changed = true
			}
		}
		for b, u := range unused {
			if b.err != "" {
				continue
			}
			if err := b.useUnused(u); err != nil {
				b.err = err.Error()
			}
			changed = true
		}
		if !changed {
			return fmt.Errorf("could not fix the type errors: %v", errs)
		}
	}
	if err := os.WriteFile(output, src, 0666); err != nil {
		return err
	}
	n := 0
	for _, b := range blocks {
		if b.err == "" {
			n++
		} else {
			fmt.Fprintf(os.Stderr, "contentcode: %s:%d: left out: %s\n", b.file, b.line, b.err)
		}
	}
	fmt.Fprintf(os.Stderr, "contentcode: compiled %d of %d code blocks into %s\n", n, len(blocks), output)
	return nil
}

// pathName returns the package name of an import path (its last element).
func pathName(path string) string {
	return path[strings.LastIndex(path, "/")+1:]
}

// packageName returns the name of the package in the current directory.
func packageName(tags string) (string, error) {
	args := []string{"list", "-f", "{{.Name}}"}
	if tags != "" {
		args = append(args, "-tags="+tags)
	}
	out, err := exec.Command("go", append(args, ".")...).Output()
	if err != nil {
		return "", fmt.Errorf("go list: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

var fenceStart = regexp.MustCompile("^(\\s*)```Go\\s*$")

// readBlocks returns the ```Go code blocks of the markdown files in dir,
// once each (by key), in file order.
func readBlocks(dir string) ([]*block, error) {
	var blocks []*block
	seen := map[string]bool{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".md" {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lines := strings.Split(string(b), "\n")
		for i := 0; i < len(lines); i++ {
			m := fenceStart.FindStringSubmatch(lines[i])
			if m == nil {
				continue
			}
			indent := m[1]
			start := i + 1
			j := start
			for ; j < len(lines); j++ {
				if strings.HasPrefix(strings.TrimSpace(lines[j]), "```") {
					break
				}
			}
			code := make([]string, 0, j-start)
			for _, l := range lines[start:j] {
				code = append(code, strings.TrimPrefix(l, indent))
			}
			src := strings.Join(code, "\n") + "\n"
			key := codeKey(src)
			if !seen[key] && strings.TrimSpace(src) != "" {
				seen[key] = true
				blocks = append(blocks, &block{file: path, line: start + 1, src: src, key: key})
			}
			i = j
		}
		return nil
	})
	return blocks, err
}

// codeKey is [cogentcore.org/core/content.CodeKey].
func codeKey(src string) string {
	s := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, src)
	h := uint32(2166136261) // FNV-1a
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return fmt.Sprintf("%08x-%d", h, len(s))
}

// generate makes the Go declarations of the block.
func (b *block) generate(importPaths map[string]string) error {
	b.pkgs = map[string]bool{}
	// a program has a main function (like yaegi, which runs it); other code
	// is a list of statements, with the parent widget as b
	text := b.src
	if !strings.HasPrefix(strings.TrimSpace(text), "package ") {
		text = "package main\n\n" + text
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "block.go", text, parser.ParseComments)
	program := err == nil && file.Scope.Lookup("main") != nil
	if !program {
		// b is in an outer block, so that the code can declare its own b
		text = "package main\n\nfunc " + b.name + "(" + parentVar + " *core.Frame, " + outVar + " io.Writer) {\nb := " + parentVar + "\n_ = b\n{\n" + b.src + "\n}\n}\n"
		fset = token.NewFileSet()
		file, err = parser.ParseFile(fset, "block.go", text, parser.ParseComments)
		if err != nil {
			return err
		}
	}
	local := map[string]string{}
	for _, im := range file.Imports {
		p, _ := strconv.Unquote(im.Path.Value)
		name := pathName(p)
		if im.Name != nil {
			name = im.Name.Name
		}
		local[name] = p
	}
	file.Imports = nil
	file.Decls = slices.DeleteFunc(file.Decls, func(d ast.Decl) bool {
		g, ok := d.(*ast.GenDecl)
		return ok && g.Tok == token.IMPORT
	})

	// identifiers that are not references: selectors and struct literal keys
	notRef := map[*ast.Ident]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			notRef[n.Sel] = true
		case *ast.KeyValueExpr:
			if id, ok := n.Key.(*ast.Ident); ok && id.Obj == nil {
				notRef[id] = true
			}
		}
		return true
	})
	top := map[*ast.Object]bool{}
	if program {
		for _, obj := range file.Scope.Objects {
			top[obj] = true
		}
	}
	// rewrite: package uses, fmt printing, top-level names, the parent b
	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			if sel, ok := n.Fun.(*ast.SelectorExpr); ok {
				if x, ok := sel.X.(*ast.Ident); ok && x.Obj == nil && x.Name == "fmt" {
					switch sel.Sel.Name {
					case "Print", "Println", "Printf":
						sel.Sel.Name = "F" + sel.Sel.Name
						n.Args = append([]ast.Expr{ast.NewIdent(outVar)}, n.Args...)
						b.pkgs["io"] = true
					}
				}
			}
		case *ast.SelectorExpr:
			if x, ok := n.X.(*ast.Ident); ok && x.Obj == nil {
				if p, ok := local[x.Name]; ok {
					b.pkgs[p] = true
				} else if p, ok := importPaths[x.Name]; ok {
					b.pkgs[p] = true
				}
			}
		case *ast.Ident:
			if notRef[n] {
				break
			}
			if n.Obj != nil && top[n.Obj] {
				n.Name = b.name + "_" + n.Name
			} else if program && n.Obj == nil && n.Name == "b" {
				n.Name = parentVar
			}
		}
		return true
	})
	if !program {
		b.pkgs["cogentcore.org/core/core"] = true
		b.pkgs["io"] = true
	}
	var buf bytes.Buffer
	for _, d := range file.Decls {
		if err := printer.Fprint(&buf, fset, &printer.CommentedNode{Node: d, Comments: file.Comments}); err != nil {
			return err
		}
		buf.WriteString("\n\n")
	}
	if program {
		fmt.Fprintf(&buf, "func %s(b *core.Frame, out io.Writer) {\n\t%s, %s = b, out\n\t%s_main()\n}\n\n", b.name, parentVar, outVar, b.name)
		b.pkgs["cogentcore.org/core/core"] = true
		b.pkgs["io"] = true
	}
	code, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}
	b.code = string(code)
	return nil
}

// useUnused inserts `_ = x` after the statements that declare the unused
// variables x of the block, given by the line of the error (relative to the
// start of the block's code: 1 is its first line).
func (b *block) useUnused(unused map[int][]string) error {
	text := "package main\n" + b.code
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "block.go", text, parser.ParseComments)
	if err != nil {
		return err
	}
	ast.Inspect(file, func(n ast.Node) bool {
		blk, ok := n.(*ast.BlockStmt)
		if !ok {
			return true
		}
		var list []ast.Stmt
		for _, st := range blk.List {
			list = append(list, st)
			rel := fset.Position(st.Pos()).Line - 1
			for _, name := range unused[rel] {
				list = append(list, &ast.AssignStmt{Lhs: []ast.Expr{ast.NewIdent("_")}, Tok: token.ASSIGN, Rhs: []ast.Expr{ast.NewIdent(name)}})
			}
		}
		blk.List = list
		return true
	})
	var buf bytes.Buffer
	for _, d := range file.Decls {
		if err := printer.Fprint(&buf, fset, &printer.CommentedNode{Node: d, Comments: file.Comments}); err != nil {
			return err
		}
		buf.WriteString("\n\n")
	}
	code, err := format.Source(buf.Bytes())
	if err != nil {
		return err
	}
	b.code = string(code)
	return nil
}

// assemble returns the output file of the blocks that are not left out,
// and sets their line ranges.
func assemble(pkgName string, blocks []*block, importPaths map[string]string) []byte {
	pkgs := map[string]bool{"cogentcore.org/core/content": true, "cogentcore.org/core/core": true, "io": true}
	for _, b := range blocks {
		if b.err != "" {
			continue
		}
		for p := range b.pkgs {
			pkgs[p] = true
		}
	}
	paths := make([]string, 0, len(pkgs))
	for p := range pkgs {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	var buf bytes.Buffer
	buf.WriteString("// Code generated by \"contentcode\"; DO NOT EDIT.\n\n")
	fmt.Fprintf(&buf, "package %s\n\nimport (\n", pkgName)
	for _, p := range paths {
		fmt.Fprintf(&buf, "\t%q\n", p)
	}
	buf.WriteString(")\n\n")
	fmt.Fprintf(&buf, "// %s and %s are the parent widget (b) and output of the\n// code block programs that are running.\nvar (\n\t%s *core.Frame\n\t%s io.Writer\n)\n\n", parentVar, outVar, parentVar, outVar)
	// an init function, not a blank variable: GoScript runs the var
	// initializers of a file of package main only if the main file imports it
	fmt.Fprintf(&buf, "func init() {\n\tcontent.AddCompiledCode(map[string]content.CodeFunc{\n")
	for _, b := range blocks {
		if b.err == "" {
			fmt.Fprintf(&buf, "\t\t%q: %s, // %s:%d\n", b.key, b.name, filepath.ToSlash(b.file), b.line)
		}
	}
	buf.WriteString("\t})\n}\n\n")
	line := bytes.Count(buf.Bytes(), []byte("\n")) + 1
	for _, b := range blocks {
		b.start, b.end = 0, 0
		if b.err != "" {
			continue
		}
		b.start = line
		fmt.Fprintf(&buf, "// %s:%d\n", filepath.ToSlash(b.file), b.line)
		buf.WriteString(b.code)
		buf.WriteString("\n")
		line += 1 + strings.Count(b.code, "\n") + 1
		b.end = line - 1
	}
	var left []string
	for _, b := range blocks {
		if b.err != "" {
			left = append(left, fmt.Sprintf("//   %s:%d: %s", filepath.ToSlash(b.file), b.line, strings.ReplaceAll(b.err, "\n", " ")))
		}
	}
	if len(left) > 0 {
		buf.WriteString("// Code blocks left out:\n" + strings.Join(left, "\n") + "\n")
	}
	return buf.Bytes()
}

// typeError is a type error at a line of the output file.
type typeError struct {
	line int
	msg  string
}

func (e typeError) String() string { return fmt.Sprintf("%d: %s", e.line, e.msg) }

// typeCheck type checks the current package with src as the output file,
// by building it with all errors reported.
func typeCheck(output string, src []byte, tags string) ([]typeError, error) {
	if err := os.WriteFile(output, src, 0666); err != nil {
		return nil, err
	}
	args := []string{"build", "-o", os.DevNull, "-gcflags=-e"}
	if tags != "" {
		args = append(args, "-tags="+tags)
	}
	cmd := exec.Command("go", append(args, ".")...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil, nil
	}
	base := filepath.Base(output)
	var errs []typeError
	for _, l := range strings.Split(string(out), "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		file, rest, _ := strings.Cut(l, ":")
		if filepath.Base(file) != base {
			return nil, fmt.Errorf("error outside of the generated file: %s", l)
		}
		ls, rest, _ := strings.Cut(rest, ":")
		_, msg, _ := strings.Cut(rest, ": ")
		line, _ := strconv.Atoi(ls)
		errs = append(errs, typeError{line: line, msg: msg})
	}
	if len(errs) == 0 {
		return nil, fmt.Errorf("go build: %v: %s", err, out)
	}
	return errs, nil
}

// blockAt returns the block at the given line of the output file.
func blockAt(blocks []*block, line int) *block {
	for _, b := range blocks {
		if b.start > 0 && line >= b.start && line <= b.end {
			return b
		}
	}
	return nil
}
