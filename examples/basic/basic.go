// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"

	"goki.dev/colors"

	"github.com/muesli/termenv"
)

func main() {
	restoreConsole, err := termenv.EnableVirtualTerminalProcessing(termenv.DefaultOutput())
	if err != nil {
		panic(err)
	}
	defer restoreConsole()

	p := termenv.ColorProfile()
	colors.SetScheme(termenv.HasDarkBackground())

	fmt.Println(termenv.String("Primary").Foreground(p.FromColor(colors.Scheme.Primary.Base)))
	fmt.Println(termenv.String("Secondary").Foreground(p.FromColor(colors.Scheme.Secondary.Base)))
	fmt.Println(termenv.String("Tertiary").Foreground(p.FromColor(colors.Scheme.Tertiary.Base)))
}
