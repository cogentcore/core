/* Copyright (c) 2018, The gide / GoKi Authors. All rights reserved. */
/* Use of this source code is governed by a BSD-style
license that can be found in the LICENSE file. */

/* This has /* embedded */ comments which is /* a bit  */ tricky */ 

func main() {
	if peas++; this > that {
		fmt.Printf("test")
		break
	}
}

/*
package main

import "github.com/goki/gi/gi"

import (
	"github.com/goki/gi/gi"
	main "github.com/goki/gi/gimain"
	"github.com/goki/gi/oswin"
	gogide "github.com/goki/gide/gide"
	"github.com/goki/pi"
	"github.com/goki/pi/piv"
)
*/
/*
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
*/
/*
type MyFloat float64

type AStruct struct {
	AField int
	TField gi.float64 `desc:"tagged"`
	AField []string
	MField map[string]int
}

var Typeo int
*/

/*
var ExprVar = "testofit"

var ExprTypeVar map[string]string 

var ExprInitMap = map[string]string{
	"does": {Val: "this work?", Bad: "dkfa"},
}
*/

/*
var ExprSlice = abc[20]

var ExprSlice2 = abc[20:30]

var ExprSlice3 = abc[20:30] + abc[:] + abc[20:] + abc[:30] + abc[20:30:2]

var ExprSelect = abc.Def
*/

/*
var ExprTypeAssert = absfr.(cheeze)

var ExprTypeAssertPtr = absfr.(*cheeze)

var ExprCvt = int(abc) // parses as a method because it doesn't know from type names

var TypPtr *Fred

var ExprCvt2 = map[string]string(ab)
*/
/*
var tree = map[token.Tokens]struct{}(optMap)

var tree = (map[token.Tokens]struct{})(optMap)

var partyp = (*int)(tree)
*/

//var methexpr = abc.meth(a-b * 2 + bf.Meth(22 + 55) / long.meth.Call(tree))

/*
// var ExprMeth = abc.meth(c)

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
}
*/

//func mainrun() {
/*
	oswin.TheApp.SetName("pie")
	oswin.TheApp.SetAbout(`<code>Pie</code> is the interactive parser (pi) editor written in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki/pi">Gide on GitHub</a> and <a href="https://github.com/goki/pi/wiki">Gide wiki</a> for documentation.<br>
<br>
Version: ` + pi.VersionInfo())
*/
/*
	if peas++; this > that {
		fmt.Printf("test")
		break
	}
*/
/*
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
*/

/*	oswin.TheApp.SetQuitCleanFunc(func() {
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
	 			ext := strings.ToLower(filepath.Ext(flag.Arg(0)))
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
*/
// }
