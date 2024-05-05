// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"

	"cogentcore.org/core/base/logx"
	"cogentcore.org/core/shell/interpreter"
)

func main() {
	logx.UserLevel = slog.LevelWarn
	in := interpreter.NewInterpreter()
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		in.Eval(line)
	}
}
