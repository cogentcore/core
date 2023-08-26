// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"

	"github.com/goki/gear"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

type App struct {
	Name    string
	Age     int
	LikesGo bool
}

var TheApp App

var TypeApp = kit.Types.AddType(&App{}, AppProps)

var AppProps = ki.Props{
	"ToolBar": ki.PropSlice{
		{"BuildCmd", ki.Props{
			"label": "Build",
		}},
	},
}

func main() {
	gear.Config(&TheApp, os.Args[1:]...)
	gear.GUI(&TheApp)
}

func (a *App) BuildCmd(output string) error {
	fmt.Println("Building")
	return nil
}
