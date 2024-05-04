// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"cogentcore.org/core/shell/parse"
)

func main() {
	ctx := parse.NewContext()
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		parse.ParseLine(ctx, line)
	}
}
