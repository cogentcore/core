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

var candidates []candidate

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
	ls := start - token.Pos(pos.Column)
	lpt := src[ls:start]
	// don't complete inside comment
	if strings.Contains(lpt, "//") {
		return completions, true // stop
	}

	end := start
	path, _ := astutil.PathEnclosingInterval(f, start, end)

	var stop bool = false // suggestion to caller about continuing to try completing
	var inBlock = false   // are we in a block (i.e. within braces)
	next := true
	for i := 0; i < len(path) && next == true; i++ {
		n := path[i]
		//fmt.Printf("%d\t%T\n", i, n)
		switch n.(type) {
		case *ast.BlockStmt:
			_, ok := n.(*ast.BlockStmt)
			if ok {
				inBlock = true // return stop
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
			if !inBlock {
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

func FindCandidateString(str string) *candidate {
	if candidates == nil {
		return nil
	}
	for _, c := range candidates {
		if c.Name == str {
			return &c
		}
	}
	return nil
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
			candidates = append(candidates, aCandidate)
			c := Completion{Text: aCandidate.Name, Icon: icon}
			c.Extra = make(map[string]string)
			c.Extra["class"] = aCandidate.Class
			c.Extra["type"] = aCandidate.Typ
			c.Extra["pkg"] = aCandidate.Pkg
			completions = append(completions, c)
		}
	}
	return completions
}

// CompleteGo is the function for completing Go code
func CompleteGo(bytes []byte, pos token.Position) []Completion {
	candidates = candidates[:0]
	var results []Completion
	var stop = false
	//results, stop = FirstPass(bytes, pos)
	if !stop && len(results) == 0 {
		results = SecondPass(bytes, pos)
	}
	return results
}

// EditGoCode is a chance to modify the completion selection before it is inserted
func EditGoCode(text string, cp int, completion Completion, seed string) (ed EditData) {
	// if the original is ChildByName() and the cursor is between d and B and the completion is Children,
	// then delete the portion after "Child" and return the new completion and the number or runes past
	// the cursor to delete
	s2 := text[cp:]
	if len(s2) > 0 {
		r := rune(s2[0])
		// find the next whitespace or end of text
		if !(unicode.IsSpace(r)) {
			count := len(s2)
			for i, c := range s2 {
				r = rune(c)
				if unicode.IsSpace(r) || r == rune('(') || r == rune('.') || r == rune('[') {
					s2 = s2[0:i]
					break
				}
				// might be last word
				if i == count-1 {
					break
				}
			}
		}
	}

	var new = completion.Text
	// todo: only do if parens not already there
	//class, ok := completion.Extra["class"]
	//if ok && class == "func" {
	//	new = new + "()"
	//}
	ed.NewText = new
	ed.ForwardDelete = len(s2)
	return ed
}
