/* Copyright (c) 2018, The gide / GoKi Authors. All rights reserved. */
/* Use of this source code is governed by a BSD-style
license that can be found in the LICENSE file. */

/* This has /* embedded */ comments which is /* a bit  */ tricky */ 

/*
var a,b,c,d = 32
var e = 22
*/

type Rule struct {
	ki.Node
	Off       bool     `desc:"disable this rule -- useful for testing"`
}

func (pr *Rule) BaseIface() reflect.Type {
	return reflect.TypeOf((*Parser)(nil)).Elem()
}

func (pr *Rule) AsParseRule() *Rule {
	return pr.This().Embed(KiT_Rule).(*Rule)
}

func tst() {
	oswin.TheApp.SetQuitCleanFunc(func() {
		fmt.Printf("Doing final Quit cleanup here..\n")
	})
}

func (pr *Parser) LexErrString() string {
	return pr.LexState.Errs.AllString()
}

func tst() {
	a = !pr.LexState.AtEol()

	pr.LexState.Filename = !pr.LexState.AtEol()

	if !pr.Sub.LexState.AtEol() && cpos == pr.LexState.Pos {
		msg := fmt.Sprintf("did not advance position -- need more rules to match current input: %v", string(pr.LexState.Src[cpos:]))
		pr.LexState.Error(cpos, msg)
		return nil
	}
}

var ext = strings.ToLower(filepath.Ext(flag.Arg(0)))

func tst() {
		if path == "" && proj == "" {
			if flag.NArg() > 0 {
	 			ext := strings.ToLower(filepath.Ext(flag.Arg(0)))
				if ext == ".gide" {
	 				proj = flag.Arg(0)
				} else {
	 				path = flag.Arg(0)
	 			}
	 		}
		}
	recv := gi.Node2DBase{}
}

func (pr *Parser) Init() {
	pr.Lexer.InitName(&pr.Lexer, "Lexer")
}

func (pr *Parser) Init2(a int, fname string, amap map[string]string) bool {
	pr.Parser.InitName(&pr.Parser, "Parser")
}

func (pr *Parser) Init3(a, b int, fname string) (bool, string) {
	pr.Ast.InitName(&pr.Ast, "Ast")
}

func (pr *Parser) Init4(a, b int, fname string) (ok bool, name string) {
	pr.LexState.Init()
}

// SetSrc sets source to be parsed, and filename it came from
func (pr *Parser) SetSrc(src [][]rune, fname string) {
}

func main() {

	if this > that {
		break
	} else {
		continue
	}

	if peas++; this > that {
		fmt.Printf("test")
		break
	}

	if this > that {
		break
	} else if something == other {
		continue
	}

	if a := b; b == a {
		fmt.Printf("test")
		break
	} else {
		continue
	}

	if a > b {
		b++
	}

	switch vvv := av; nm {
	case "baby":
		nm = "maybe"
		a++
	case "not":
		i++
		p := a * i
	default:
		non := "anon"
	}

	for i := 0; i < 100; i++ {
		fmt.Printf("%v %v", a, i)
		a++
		p := a * i
	}

	for a, i, j, k := range names {
		for i < 100 {
			for {
				fmt.Printf("%v %v", a, i)
			}
		}
	}

	fmt.Printf("starting test")
	defer my.Widget.UpdateEnd(updt)
	goto bypass

	if a == b {
		fmt.Printf("equal")
	} else if a > b {
		so++
		be--
		it := 20
	} else {
		fmt.Printf("long one")
	}

bypass:
	fmt.Printf("here")
	return
	return something
	{
		nvar := 22
		nvar += function(of + some + others)
	}
}

package main

import "github.com/goki/gi/gi"

import (
	gi "github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/oswin"
	gogide "github.com/goki/gide/gide"
	"github.com/goki/pi"
	"github.com/goki/pi/piv"
)

const neg = -1

const neg2 = -(2+2)

const Prec1 = ((2-1) * 3)

const (
	parn = 1 + (2 + 3)
	PrecedenceS2 = 25 / (3.14 + -(2 + 4)) > ((2 - 5) * 3)
)

const Precedence2 = -(3.14 + 2 * 4)

// The lexical acts
const (
	// Next means advance input position to the next character(s) after the matched characters
	Next Actions = 4.92

	// Name means read in an entire name, which is letters, _ and digits after first letter
	// position will be advanced to just after
	Name
)

type MyFloat float64

type AStruct struct {
	AField int
	TField gi.Widget `desc:"tagged"`
	AField []string
	MField map[string]int
}

var Typeo int

var ExprVar = "testofit"

var ExprTypeVar map[string]string 

var ExprInitMap = map[string]string{
	"does": {Val: "this work?", Bad: "dkfa"},
}

var ExprSlice = abc[20]

var ExprSlice2 = abc[20:30]

var ExprSlice3 = abc[20:30] + abc[:] + abc[20:] + abc[:30] + abc[20:30:2]

var ExprSelect = abc.Def

var ExprCvt = int(abc) // works with basic type names -- others are FunCall

var TypPtr *Fred

var ExprCvt2 = map[string]string(ab)

var tree = map[token.Tokens]struct{}(optMap)

var tree = (map[token.Tokens]struct{})(optMap)

var partyp = (*int)(tree)

var ExprTypeAssert = absfr.(gi.TreeView)

var ExprTypeAssertPtr = absfr.(*gi.TreeView)

var methexpr = abc.meth(a-b * 2 + bf.Meth(22 + 55) / long.meth.Call(tree))

var ExprMeth = abc.meth(c)

var ExprMethLong = long.abc.meth(c)

var ExprFunNil = fun()

var ExprFun = meth(2 + 2)

var ExprFun = meth(2 + 2, fslaf)

var ExprFunElip = meth(2 + 2, fslaf...)


func main() {
	a <- b
	c++
	c[3] = 42 * 17
 	bf := a * b + c[32]
	d += funcall(a, b, c...)
	fmt.Printf("this is ok", gi.CallMe(a.(tree) + b.(*tree) + int(22) * string(17)))
}

func mainrun() {
	oswin.TheApp.SetName("pie")
	oswin.TheApp.SetAbout(`<code>Pie</code> is the interactive parser (pi) editor written in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki/pi">Gide on GitHub</a> and <a href="https://github.com/goki/pi/wiki">Gide wiki</a> for documentation.<br>
<br>
Version: ` + pi.VersionInfo())
	if peas++; this > that {
		fmt.Printf("test")
		break
	}
	if this > that {
		fmt.Printf("test")
		break
	} else {
		continue
	}

	if this > that {
		fmt.Printf("test")
		break
	} else if something == other {
		continue
	}

	oswin.TheApp.SetQuitCleanFunc(func() {
		fmt.Printf("Doing final Quit cleanup here..\n")
	})

	pi.InitPrefs()

	var path string
	var proj string
	// process command args
	if len(os.Args) > 1 {
		flag.StringVar(&path, "path", "", "path to open -- can be to a directory or a filename within the directory")
		flag.StringVar(&proj, "proj", "", "project file to open -- typically has .gide extension")
		// todo: other args?
		flag.Parse()
		if path == "" && proj == "" {
			if flag.NArg() > 0 {
	 			ext = strings.ToLower(filepath.Ext(flag.Arg(0)))
				if ext == ".gide" {
	 				proj = flag.Arg(0)
				} else {
	 				path = flag.Arg(0)
	 			}
	 		}
		}
	}

	recv := gi.Node2DBase{}
	recv.InitName(&recv, "pie_dummy")

	inQuitPrompt := false
	oswin.TheApp.SetQuitReqFunc(func() {
		if !inQuitPrompt {
			inQuitPrompt = true
			if gide.QuitReq() {
				oswin.TheApp.Quit()
			} else {
				inQuitPrompt = false
			}
		}
	})

	if proj != "" {
		proj, _ = filepath.Abs(proj)
	 	gide.OpenGideProj(proj)
	} else {
		if path != "" {
			path, _ = filepath.Abs(path)
		}
		gide.NewGideProjPath(path)
	}

	piv.NewPiView()

	// above NewGideProj calls will have added to WinWait..
	gi.WinWait.Wait()
}

var someother int

