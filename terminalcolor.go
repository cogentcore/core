// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goosi

import (
	"fmt"
	"image/color"
	"io"
	"os"
	"os/exec"
	"strings"
)

// Based on:
// https://github.com/Canop/terminal-light
// https://invisible-island.net/xterm/ctlseqs/ctlseqs.html
// https://stackoverflow.com/questions/28096697/how-to-get-current-terminal-color-pair-in-bash/28334701#28334701

// TerminalColor returns the background color of the current terminal.
func TerminalColor() (color.RGBA, error) {
	cmd := exec.Command("printf", "\x1b]11;?\x07")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return color.RGBA{}, fmt.Errorf("error running command to get terminal color: %w; output: %s", err, out)
	}
	si := ""
	fmt.Println(io.ReadAll(os.Stdin))
	fmt.Println("si", si)
	fmt.Println(out)
	s := string(out)
	fmt.Printf("init %q\n", s)
	s = strings.TrimPrefix(s, "\x1b]")
	s = strings.TrimPrefix(s, "11;rgb:")
	fmt.Printf("after %q\n", s)
	fmt.Println(strings.Contains(s, "1818"))
	return color.RGBA{}, nil
}
