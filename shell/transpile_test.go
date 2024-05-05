// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type exIn struct {
	i string
	e string
}

func TestTranspile(t *testing.T) {
	tests := []exIn{
		{"`ls -la`\n", `shell.Exec("ls", "-la")`},
		{`var name string`, `var name string`},
		{`name = "test"`, `name = "test"`},
		{`echo {name}`, `shell.Exec("echo", name)`},
		{`echo "testing"`, `shell.Exec("echo", "testing")`},
		{`number := 1.23`, `number := 1.23`},
		{`for i := 0; i < 3; i++ { fmt.Println(i, "\n")`, `for i := 0; i < 3; i++ { fmt.Println(i, "\n")`},
		{"for i, v := range `ls -la` {", `for i, v := range shell.Output("ls", "-la") {`},
		{`// todo: fixit`, `// todo: fixit`},
	}

	sh := NewShell()
	for _, test := range tests {
		o := sh.TranspileLine(test.i)
		assert.Equal(t, test.e, o)
	}
}
