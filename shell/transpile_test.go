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

// these are more general tests of full-line statements of various forms
func TestTranspile(t *testing.T) {
	logx.UserLevel = slog.LevelDebug
	tests := []exIn{
		// {"`ls -la`", `shell.Exec("ls", "-la")`},
		// {`var name string`, `var name string`},
		// {`name = "test"`, `name = "test"`},
		// {`echo {name}`, `shell.Exec("echo", name)`},
		// {`echo "testing"`, `shell.Exec("echo", "testing")`},
		// {`number := 1.23`, `number := 1.23`},
		// {`println("hi")`, `println("hi")`},
		// {`fmt.Println("hi")`, `fmt.Println("hi")`},
		// {`for i := 0; i < 3; i++ { fmt.Println(i, "\n")`, `for i := 0; i < 3; i++ { fmt.Println(i, "\n")`},
		// {"for i, v := range `ls -la` {", `for i, v := range shell.Output("ls", "-la") {`},
		// {`// todo: fixit`, `// todo: fixit`},
		{"`go build`", `shell.Exec("go", "build")`},
		{"{go build()}", `go build()`},
		{"go build", `shell.Exec("go", "build")`},
		{"go build()", `go build()`},
		{"set something hello-1", `shell.Exec("set", "something", "hello-1")`},
		{"set something = hello", `shell.Exec("set", "something = hello")`},
		{`set "something=hello"`, `shell.Exec("set", "something=hello")`},
		{`set something="hello"`, `shell.Exec("set", "something=hello")`},
	}

	sh := NewShell()
	for _, test := range tests {
		o := sh.TranspileLine(test.i)
		assert.Equal(t, test.e, o)
	}
}

// Paths tests focus specifically on the Path() and ExecIdent() code
// in paths.go
func TestPaths(t *testing.T) {
	// logx.UserLevel = slog.LevelDebug
	tests := []exIn{
		{`fmt.Println("hi")`, `fmt.Println("hi")`},
		{"cosh -i", `shell.Exec("cosh", "-i")`},
		{"./cosh -i", `shell.Exec("./cosh", "-i")`},
		{"cat main.go", `shell.Exec("cat", "main.go")`},
		{"cd cogent", `shell.Exec("cd", "cogent")`},
		{"cd cogent/", `shell.Exec("cd", "cogent/")`},
		{"echo $PATH", `shell.Exec("echo", "$PATH")`},
		{`"./Cogent Code"`, `shell.Exec("./Cogent Code")`},
		{`Cogent\ Code`, `shell.Exec("Cogent Code")`},
		{`./Cogent\ Code`, `shell.Exec("./Cogent Code")`},
		{`ios\ deploy -i`, `shell.Exec("ios deploy", "-i")`},
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
