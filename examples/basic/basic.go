// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"goki.dev/colors"
	"goki.dev/matcolor"

	"github.com/muesli/termenv"
)

func main() {
	restoreConsole, err := termenv.EnableVirtualTerminalProcessing(termenv.DefaultOutput())
	if err != nil {
		panic(err)
	}
	defer restoreConsole()

	matcolor.TheSchemes = matcolor.NewSchemes(matcolor.NewPalette(matcolor.KeyFromPrimary(colors.MustFromHex("#4285F4"))))

	p := termenv.ColorProfile()
	if termenv.HasDarkBackground() {
		matcolor.TheScheme = &matcolor.TheSchemes.Dark
	} else {
		matcolor.TheScheme = &matcolor.TheSchemes.Light
	}

	fmt.Print(termenv.String("Hello, World").Foreground(p.FromColor(matcolor.TheScheme.Primary)))
}
