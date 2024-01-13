// Copyright 2023 The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"image"

	"goki.dev/goosi"
	_ "goki.dev/goosi/driver"
)

func main() {
	fmt.Println("Hello, world!")
	opts := &goosi.NewWindowOptions{
		Size:      image.Pt(1024, 768),
		StdPixels: true,
		Title:     "Goosi Test Window",
	}
	w, err := goosi.TheApp.NewWindow(opts)
	if err != nil {
		panic(err)
	}

	fmt.Println("got new window", w)
	goosi.TheApp.MainLoop()
}
