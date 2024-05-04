// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

func TestSymbols(t *testing.T) {
	t.Skip("nn")
	i := interp.New(interp.Options{})
	i.Use(stdlib.Symbols)
	// https://github.com/traefik/yaegi/issues/1629
	syms := i.Symbols("fmt") // using "" causes carash
	fmt.Println(syms)
}

func TestParse(t *testing.T) {
	tests := []string{
		`var name string
name = "test"
echo name
`,
		`name := "test"
echo name
`,
		`number := 1.23
echo number
`,
		`for i := 0; i < 3; i++ { print(i, "\n") }
echo i
`, // todo: following doesn't work for unknown reasons
		`for i := 0; i < 3; i++ { fmt.Println(i) }
echo i
`, // todo: following doesn't work b/c yaegi won't process just open brace
		`for i := 0; i < 3; i++ {
	echo i
}
`,
	}

	for ti, test := range tests {
		fmt.Println("\n########## Test: ", ti)
		sh := NewShell()
		// sh.SetDebug(1)
		reader := bufio.NewReader(bytes.NewBufferString(test))
		for {
			line, err := reader.ReadString('\n')
			if err == io.EOF {
				break
			}
			sh.ParseLine(line)
		}
	}
}
