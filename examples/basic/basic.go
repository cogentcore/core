// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"image"

	"goki.dev/goosi"
	"goki.dev/goosi/driver"
	"goki.dev/grr"
)

func main() {
	fmt.Println("Hello, basic!")
	driver.Main(mainrun)
}

func mainrun(a goosi.App) {
	fmt.Println("mainrun")
	// time.Sleep(5 * time.Second)
	opts := &goosi.NewWindowOptions{
		Size:      image.Pt(1024, 768),
		StdPixels: true,
		Title:     "Goosi Test Window",
	}
	w := grr.Log1(goosi.TheApp.NewWindow(opts))

	fmt.Println("got new window", w)

	select {}
}
