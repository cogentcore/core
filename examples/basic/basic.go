// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"

	"github.com/goki/gear"
)

type App struct {
	Name    string
	Age     int
	LikesGo bool
}

var TheApp App

func main() {
	gear.Config(&TheApp, os.Args[1:]...)
	gear.GUI(&TheApp)
}
