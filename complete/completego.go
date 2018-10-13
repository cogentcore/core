package complete

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os/exec"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/tools/go/ast/astutil"
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

//func CompleteGo(bytes []byte, pos token.Position) (matches []string, seed string) {
//	Init()
//	src := string(bytes)
//	matches = matches[:0]
//	fs := token.NewFileSet()
//	f, err := parser.ParseFile(fs, "src.go", src, parser.AllErrors)
//	if err != nil {
//		log.Printf("could not parse %s: %v\n", f, err)
//	}
//
//	ast.Walk(v, f)
//	start := token.Pos(pos.Offset)
//	end := start
//	path, _ := astutil.PathEnclosingInterval(f, start, end)
//	lineUpToPos := src[pos.Offset-pos.Column : pos.Offset]
//
//	next := true
//	for i := 0; i < len(path) && next == true; i++ {
//		n := path[i]
//		//fmt.Printf("%d\t%T\n", i, n)
//		switch n := n.(type) {
//		case *ast.BadDecl:
//			fmt.Printf("\t%T.Doc\n", n)
//			if i+1 < len(path) {
//				n2 := path[i+1]
//				switch n2.(type) {
//				case *ast.File:
//					matches = append(matches, decls...)
//					next = false
//				}
//			}
//			if strings.Contains(lineUpToPos, ":=") { // must be a better way
//				matches = append(matches, Locals()...)
//				matches = append(matches, Funcs()...)
//				next = false
//			}
//
//		case *ast.Ident:
//			fmt.Printf("\t%T.Doc: %q\n", n, n.Name)
//			matches = append(matches, Locals()...)
//			matches = append(matches, Funcs()...)
//			next = false
//		case *ast.Field:
//			fmt.Printf("\t%T.Doc: %q\n", n, n.Doc.Text())
//		case *ast.ImportSpec:
//			fmt.Printf("\t%T.Doc: %q\n", n, n.Doc.Text())
//		case *ast.ValueSpec:
//			fmt.Printf("\t%T.Doc: %q\n", n, n.Doc.Text())
//		case *ast.GenDecl:
//			fmt.Printf("\t%T.Doc: %q\n", n, n.Doc.Text())
//			if n.Tok == token.IMPORT {
//				next = false
//			}
//		case *ast.TypeSpec:
//			fmt.Printf("\t%T.Doc: %q\n", n, n.Doc.Text())
//		case *ast.FuncDecl:
//			fmt.Printf("\t%T.Doc: %q\n", n, n.Doc.Text())
//			if !strings.HasPrefix(lineUpToPos, "func") { // can't guess name of new function
//				matches = append(matches, Locals()...)
//				matches = append(matches, Funcs()...)
//			}
//			next = false
//		case *ast.SelectorExpr:
//			path, _ := astutil.PathEnclosingInterval(f, start-1, end)
//			n2 := path[0]
//			fmt.Printf("\t%T.Doc: %q\n", n2)
//			zType := reflect.TypeOf(n2)
//			switch zType.Kind() {
//			case reflect.Struct:
//				fmt.Println("dog")
//			}
//		case *ast.File:
//			fmt.Printf("\t%T.Doc: %q\n", n, n.Doc.Text())
//			matches = append(matches, decls...)
//			next = false
//		case *ast.BasicLit:
//			fmt.Printf("\t%T.Value: %q\n", n, n.Value)
//			if i+1 < len(path) {
//				n2 := path[i+1]
//				switch n2.(type) {
//				case *ast.ImportSpec:
//					fmt.Printf("\ttodo: package/filename completion for imports\n")
//				}
//			}
//			next = false
//		case *ast.AssignStmt:
//			fmt.Printf("\t%T.LHS: %q\n", n, n.Lhs)
//			fmt.Printf("\t%T.RHS: %q\n", n, n.Rhs)
//		}
//	}
//	seed = SeedWhiteSpace(lineUpToPos)
//	return matches, seed
//}
//
//func Locals() []string {
//	keys := reflect.ValueOf(v.locals).MapKeys()
//	locals := make([]string, len(keys))
//	for i := 0; i < len(keys); i++ {
//		locals[i] = keys[i].String()
//		//fmt.Println(locals[i])
//	}
//	sort.Strings(locals)
//	return locals
//}
//
//func Funcs() []string {
//	keys := reflect.ValueOf(v.funcs).MapKeys()
//	funcs := make([]string, len(keys))
//	for i := 0; i < len(keys); i++ {
//		funcs[i] = keys[i].String()
//		//fmt.Println(funcs[i])
//	}
//	sort.Strings(funcs)
//	return funcs
//}

// FirstPass handles some cases of completion that gocode either
// doesn't handle or doesn't do well - this will be expanded to more cases
func FirstPass(bytes []byte, pos token.Position) ([]Completion, bool) {
	var completions []Completion

	src := string(bytes)
	fs := token.NewFileSet()
	f, _ := parser.ParseFile(fs, "src.go", src, parser.AllErrors)
	//if err != nil {
	//	log.Printf("could not parse %s: %v\n", f, err)
	//}

	start := token.Pos(pos.Offset)
	linestart := start - token.Pos(pos.Column)
	linepretext := src[linestart:start]
	// don't complete inside comment
	if strings.Contains(linepretext, "//") {
		return completions, true // stop
	}

	end := start
	path, _ := astutil.PathEnclosingInterval(f, start, end)

	var stop bool = false // suggestion to caller about continuing to try completing
	var inblock = false   // are we in a block (i.e. within braces)
	next := true
	for i := 0; i < len(path) && next == true; i++ {
		n := path[i]
		//fmt.Printf("%d\t%T\n", i, n)
		switch n.(type) {
		case *ast.BlockStmt:
			_, ok := n.(*ast.BlockStmt)
			if ok {
				inblock = true // return stop
			}
		case *ast.GenDecl:
			gd, ok := n.(*ast.GenDecl)
			if ok {
				// todo: for now don't complete but we could path complete
				if gd.Tok == token.IMPORT {
					return completions, true // return stop
				}
			}
		case *ast.BadDecl:
			//fmt.Printf("\t%T.Doc\n", n)
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
			if !inblock {
				for _, aCandidate := range decls {
					comp := Completion{Text: aCandidate}
					completions = append(completions, comp)
				}
			}
			next = false
		default:
			next = false
		}
	}
	return completions, stop
}

type candidate struct {
	Class string `json:"class"`
	Name  string `json:"name"`
	Typ   string `json:"type"`
	Pkg   string `json:"package"`
}

// SecondPass uses the gocode server to find possible completions at the specified position
// in the src (i.e. the byte slice passed in)
// bytes should be the current in memory version of the file
func SecondPass(bytes []byte, pos token.Position) []Completion {
	var completions []Completion

	offset := pos.Offset
	offsetString := strconv.Itoa(offset)
	cmd := exec.Command("gocode", "-f=json", "-ignore-case", "-builtin", "autocomplete", offsetString)
	cmd.Stdin = strings.NewReader(string(bytes)) // use current state of file not disk version - may be stale
	result, _ := cmd.Output()
	var skip int = -1
	for i := 0; i < len(result); i++ {
		if result[i] == 123 { // 123 is 07b is '{'
			skip = i - 1 // stop when '{' is found
			break
		}
	}
	if skip != -1 {
		result = result[skip : len(result)-2] // strip off [N,[ at start (where N is some number) and trailing ]] as well
		data := make([]candidate, 0)
		err := json.Unmarshal(result, &data)
		if err != nil {
			fmt.Printf("%#v", err)
		}
		var icon string
		for _, aCandidate := range data {
			switch aCandidate.Class {
			case "const":
				icon = "const"
			case "func":
				icon = "func"
			case "package":
				icon = "package"
			case "type":
				icon = "type"
			case "var":
				icon = "var"
			default:
				icon = "blank"
			}
			comp := Completion{Text: aCandidate.Name, Icon: icon}
			completions = append(completions, comp)
		}
	}
	return completions
}

// CompleteGo is the function for completing Go code
func CompleteGo(bytes []byte, pos token.Position) []Completion {
	var results []Completion
	results, stop := FirstPass(bytes, pos)
	if !stop && len(results) == 0 {
		results = SecondPass(bytes, pos)
	}
	return results
}

// EditGoCode replaces the completion seed and any text up to the next whitespace or other go delimiter
// with the selected completion. delta is the change in cursor position (cp).
func EditGoCode(text string, cp int, completion string, seed string) (newText string, delta int) {
	s1 := string(text[0:cp])
	s2 := string(text[cp:])

	if len(s2) > 0 {
		r := rune(s2[0])
		// find the next whitespace or end of text
		if !(unicode.IsSpace(r)) {
			count := len(s2)
			for i, c := range s2 {
				r = rune(c)
				if unicode.IsSpace(r) || r == rune('(') || r == rune('.') || r == rune('[') {
					s2 = s2[i:]
					break
				}
				// might be last word
				if i == count-1 {
					s2 = ""
				}
			}
		}
	}

	s1 = strings.TrimSuffix(s1, seed)
	s1 += completion
	t := s1 + s2
	delta = len(completion) - len(seed)
	return t, delta
}
