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
)

func TestTranspile(t *testing.T) {
	tests := []string{
		"`ls -la`\n",
		`var name string
name = "test"
echo {name}
`,
		`name := "test"
echo {name}
`,
		`number := 1.23
echo {number}
`,
		`for i := 0; i < 3; i++ { print(i, "\n") }
echo {i}
`, // todo: following doesn't work for unknown reasons
		`for i := 0; i < 3; i++ { fmt.Println(i) }
echo {i}
`, // todo: following doesn't work b/c yaegi won't process just open brace
		`for i := 0; i < 3; i++ {
		echo {i}
}
`,
	}

	for ti, test := range tests {
		fmt.Println("\n########## Test: ", ti)
		sh := NewShell()
		reader := bufio.NewReader(bytes.NewBufferString(test))
		for {
			line, err := reader.ReadString('\n')
			if err == io.EOF {
				break
			}
			tln := sh.TranspileLine(line)
			fmt.Println("## input:\n", line, "\n## output:\n", tln)
		}
	}
}
