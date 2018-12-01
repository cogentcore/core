/* Copyright (c) 2018, The gide / GoKi Authors. All rights reserved. */
/* Use of this source code is governed by a BSD-style
license that can be found in the LICENSE file. */

/* This has /* embedded */ comments which is /* a bit  */ tricky */ 

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
const Prec1 = ((2-1) * 3)

const (
	parn = 1 + (2 + 3)
	PrecedenceS2 = 25 / (3.14 + -(2 + 4)) > ((2 - 5) * 3)
)

const Precedence2 = -(3.14 + 2 * 4)
*/

/*
type MyFloat float64

type AStruct struct {
	AField int
	TField gi.float64 `desc:"tagged"`
	AField []string
	MField map[string]int
}

// The lexical acts
const (
	// Next means advance input position to the next character(s) after the matched characters
	Next Actions = 4.92

	// Name means read in an entire name, which is letters, _ and digits after first letter
	// position will be advanced to just after
	Name
)

var Typeo int

var ExprVar = "testofit"

var ExprTypeVar map[string]string 

var ExprInitMap = map[string]string{
	"does": {Val: "this work?", Bad: "dkfa"},
}
*/

func main() {
	gimain.Main(func() {
		mainrun()
	})
}

func mainrun() {
	oswin.TheApp.SetName("pie")
	oswin.TheApp.SetAbout(`<code>Pie</code> is the interactive parser (pi) editor written in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki/pi">Gide on GitHub</a> and <a href="https://github.com/goki/pi/wiki">Gide wiki</a> for documentation.<br>
<br>
Version: ` + pi.VersionInfo())

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
}
