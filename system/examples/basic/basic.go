// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"image"

	"cogentcore.org/core/system"
	_ "cogentcore.org/core/system/driver"
)

func main() {
	fmt.Println("Hello, world!")
	opts := &system.NewWindowOptions{
		Size:      image.Pt(1024, 768),
		StdPixels: true,
		Title:     "System Test Window",
	}
	w, err := system.TheApp.NewWindow(opts)
	if err != nil {
		panic(err)
	}

	fmt.Println("got new window", w)
	system.TheApp.MainLoop()
}
