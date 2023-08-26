// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import econfig "github.com/goki/gear"

type App struct {
	Name string
	On   bool
}

var TheApp App

func main() {
	econfig.Config(&TheApp, "config.toml")
}
