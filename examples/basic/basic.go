// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

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
	err := gear.Run(&TheApp, "config.toml")
	if err != nil {
		fmt.Println(err)
	}
}

func (a *App) BuildCmd() error {
	fmt.Println("Building")
	return nil
}
