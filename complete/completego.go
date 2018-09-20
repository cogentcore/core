package complete

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"golang.org/x/tools/go/ast/astutil"
	"log"
	"reflect"
	"sort"
	"strings"
)

type visitor struct {
	locals map[string]int
	funcs  map[string]int
}

var decls = []string{"const", "func", "import", "type", "var"}
var v visitor

func (v visitor) Visit(n ast.Node) ast.Visitor {
	if n == nil {
		return nil
	}

	switch d := n.(type) {
	case *ast.AssignStmt:
		for _, name := range d.Lhs {
			if ident, ok := name.(*ast.Ident); ok {
				if ident.Name == "_" {
					//fmt.Println("no identifier")
					continue
				}
				if ident.Obj != nil && ident.Obj.Pos() == ident.Pos() {
					v.locals[ident.Name]++
				}
			}
		}
	case *ast.Ident:
		if ident, ok := n.(*ast.Ident); ok {
			if ident.Name == "_" {
				//fmt.Println("no identifier")
			}
			if ident.Obj != nil && ident.Obj.Pos() == ident.Pos() {
				v.locals[ident.Name]++
			}
		}
	case *ast.FuncDecl:
		if fun, ok := n.(*ast.FuncDecl); ok {
			v.funcs[fun.Name.Name]++
		}
	}
	return v
}

func Init() {
	v.locals = make(map[string]int)
	v.funcs = make(map[string]int)
}

func CompleteGo(bytes []byte, pos token.Position) (matches []string, seed string) {
	Init()
	src := string(bytes)
	matches = matches[:0]
	fs := token.NewFileSet()
	f, err := parser.ParseFile(fs, "src.go", src, parser.AllErrors)
	if err != nil {
		log.Printf("could not parse %s: %v\n", f, err)
	}

	ast.Walk(v, f)
	start := token.Pos(pos.Offset)
	end := start
	path, _ := astutil.PathEnclosingInterval(f, start, end)
	lineUpToPos := src[pos.Offset-pos.Column : pos.Offset]

	next := true
	for i := 0; i < len(path) && next == true; i++ {
		n := path[i]
		//fmt.Printf("%d\t%T\n", i, n)
		switch n := n.(type) {
		case *ast.BadDecl:
			fmt.Printf("\t%T.Doc\n", n)
			if i+1 < len(path) {
				n2 := path[i+1]
				switch n2.(type) {
				case *ast.File:
					matches = append(matches, decls...)
					next = false
				}
			}
			if strings.Contains(lineUpToPos, ":=") { // must be a better way
				matches = append(matches, Locals()...)
				matches = append(matches, Funcs()...)
				next = false
			}

		case *ast.Ident:
			fmt.Printf("\t%T.Doc: %q\n", n, n.Name)
			matches = append(matches, Locals()...)
			matches = append(matches, Funcs()...)
			next = false
		case *ast.Field:
			fmt.Printf("\t%T.Doc: %q\n", n, n.Doc.Text())
		case *ast.ImportSpec:
			fmt.Printf("\t%T.Doc: %q\n", n, n.Doc.Text())
		case *ast.ValueSpec:
			fmt.Printf("\t%T.Doc: %q\n", n, n.Doc.Text())
		case *ast.GenDecl:
			fmt.Printf("\t%T.Doc: %q\n", n, n.Doc.Text())
			if n.Tok == token.IMPORT {
				next = false
			}
		case *ast.TypeSpec:
			fmt.Printf("\t%T.Doc: %q\n", n, n.Doc.Text())
		case *ast.FuncDecl:
			fmt.Printf("\t%T.Doc: %q\n", n, n.Doc.Text())
			if !strings.HasPrefix(lineUpToPos, "func") { // can't guess name of new function
				matches = append(matches, Locals()...)
				matches = append(matches, Funcs()...)
			}
			next = false
		case *ast.SelectorExpr:
			path, _ := astutil.PathEnclosingInterval(f, start-1, end)
			n2 := path[0]
			fmt.Printf("\t%T.Doc: %q\n", n2)
			zType := reflect.TypeOf(n2)
			switch zType.Kind() {
			case reflect.Struct:
				fmt.Println("dog")
			}
		case *ast.File:
			fmt.Printf("\t%T.Doc: %q\n", n, n.Doc.Text())
			matches = append(matches, decls...)
			next = false
		case *ast.BasicLit:
			fmt.Printf("\t%T.Value: %q\n", n, n.Value)
			if i+1 < len(path) {
				n2 := path[i+1]
				switch n2.(type) {
				case *ast.ImportSpec:
					fmt.Printf("\ttodo: package/filename completion for imports\n")
				}
			}
			next = false
		case *ast.AssignStmt:
			fmt.Printf("\t%T.LHS: %q\n", n, n.Lhs)
			fmt.Printf("\t%T.RHS: %q\n", n, n.Rhs)
		}
	}
	seed = SeedWhiteSpace(lineUpToPos)
	return matches, seed
}

func Locals() []string {
	keys := reflect.ValueOf(v.locals).MapKeys()
	locals := make([]string, len(keys))
	for i := 0; i < len(keys); i++ {
		locals[i] = keys[i].String()
		//fmt.Println(locals[i])
	}
	sort.Strings(locals)
	return locals
}

func Funcs() []string {
	keys := reflect.ValueOf(v.funcs).MapKeys()
	funcs := make([]string, len(keys))
	for i := 0; i < len(keys); i++ {
		funcs[i] = keys[i].String()
		//fmt.Println(funcs[i])
	}
	sort.Strings(funcs)
	return funcs
}

// FirstPassComplete handles some cases of completion that gocode either
// doesn't handle or doesn't do well - this will be expanded to more cases
func FirstPassComplete(bytes []byte, pos token.Position) []Completion {
	var completions []Completion

	src := string(bytes)
	fs := token.NewFileSet()
	f, _ := parser.ParseFile(fs, "src.go", src, parser.AllErrors)
	//if err != nil {
	//	log.Printf("could not parse %s: %v\n", f, err)
	//}

	start := token.Pos(pos.Offset)
	end := start
	path, _ := astutil.PathEnclosingInterval(f, start, end)

	next := true
	for i := 0; i < len(path) && next == true; i++ {
		n := path[i]
		//fmt.Printf("%d\t%T\n", i, n)
		switch n.(type) {
		case *ast.BadDecl:
			fmt.Printf("\t%T.Doc\n", n)
			if i+1 < len(path) {
				n2 := path[i+1]
				switch n2.(type) {
				case *ast.File:
					for _, aCandidate := range decls {
						comp := Completion{Text: aCandidate}
						completions = append(completions, comp)
					}
				}
			}
			next = false
		case *ast.File:
			//fmt.Printf("\t%T.Doc: %q\n", n, n.Doc.Text())
			for _, aCandidate := range decls {
				comp := Completion{Text: aCandidate}
				completions = append(completions, comp)
			}
			next = false
		default:
			next = false
		}
	}
	return completions
}
