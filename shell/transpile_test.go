// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shell

import (
	"log/slog"
	"testing"

	"cogentcore.org/core/base/logx"
	"github.com/stretchr/testify/assert"
)

type exIn struct {
	i string
	e string
}

func TestTranspile(t *testing.T) {
	tests := []exIn{
		{"`ls -la`", `shell.Exec("ls", "-la")`},
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

func TestPaths(t *testing.T) {
	logx.UserLevel = slog.LevelDebug
	tests := []exIn{
		{"cosh -i", `shell.Exec("cosh", "-i")`},
		{"./cosh -i", `shell.Exec("./cosh", "-i")`},
		// {`ios\ deploy -i`, `shell.Exec("ios deploy", "-i")`},
		{"./ios-deploy -i", `shell.Exec("./ios-deploy", "-i")`},
		{"ios_deploy -i tree_file", `shell.Exec("ios_deploy", "-i", "tree_file")`},
		{"ios_deploy/sub -i tree_file", `shell.Exec("ios_deploy/sub", "-i", "tree_file")`},
		{"C:/ios_deploy/sub -i tree_file", `shell.Exec("C:/ios_deploy/sub", "-i", "tree_file")`},
		{"ios_deploy -i tree_file/path", `shell.Exec("ios_deploy", "-i", "tree_file/path")`},
		{"ios-deploy -i", `shell.Exec("ios-deploy", "-i")`},
		{"ios-deploy -i tree-file", `shell.Exec("ios-deploy", "-i", "tree-file")`},
		{"ios-deploy -i tree-file/path/here", `shell.Exec("ios-deploy", "-i", "tree-file/path/here")`},
		{"cd ..", `shell.Exec("cd", "..")`},
		{"cd ../another/dir/to/go_to", `shell.Exec("cd", "../another/dir/to/go_to")`},
		{"cd ../an-other/dir/", `shell.Exec("cd", "../an-other/dir/")`},
		{"curl https://google.com/search?q=hello%20world#body", `shell.Exec("curl", "https://google.com/search?q=hello%20world#body")`},
	}
	sh := NewShell()
	for _, test := range tests {
		o := sh.TranspileLine(test.i)
		assert.Equal(t, test.e, o)
	}
}
