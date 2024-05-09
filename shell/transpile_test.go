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

// these are more general tests of full-line statements of various forms
func TestTranspile(t *testing.T) {
	// logx.UserLevel = slog.LevelDebug
	tests := []exIn{
		{"`ls -la`", `shell.Run("ls", "-la")`},
		{`var name string`, `var name string`},
		{`name = "test"`, `name = "test"`},
		{`echo {name}`, `shell.Run("echo", name)`},
		{`echo "testing"`, `shell.Run("echo", "testing")`},
		{`number := 1.23`, `number := 1.23`},
		{`println("hi")`, `println("hi")`},
		{`fmt.Println("hi")`, `fmt.Println("hi")`},
		{`for i := 0; i < 3; i++ { fmt.Println(i, "\n")`, `for i := 0; i < 3; i++ { fmt.Println(i, "\n")`},
		{"for i, v := range `ls -la` {", `for i, v := range shell.Output("ls", "-la") {`},
		{`// todo: fixit`, `// todo: fixit`},
		{"`go build`", `shell.Run("go", "build")`},
		{"{go build()}", `go build()`},
		{"go build", `shell.Run("go", "build")`},
		{"go build()", `go build()`},
		{"go build &", `shell.Start("go", "build")`},
		{"[mkdir subdir]", `shell.ExecErrOK("mkdir", "subdir")`},
		{"set something hello-1", `shell.Run("set", "something", "hello-1")`},
		{"set something = hello", `shell.Run("set", "something", "=", "hello")`},
		{`set something = "hello"`, `shell.Run("set", "something", "=", "hello")`},
		{`set something=hello`, `shell.Run("set", "something=hello")`},
		{`set "something=hello"`, `shell.Run("set", "something=hello")`},
		{`set something="hello"`, `shell.Run("set", "something=\"hello\"")`},
		{`add-path /opt/sbin /opt/homebrew/bin`, `shell.Run("add-path", "/opt/sbin", "/opt/homebrew/bin")`},
		{`cat file > test.out`, `shell.Run("cat", "file", ">", "test.out")`},
		{`cat file | grep -v exe > test.out`, `shell.Run("cat", "file", "|", "grep", "-v", "exe", ">", "test.out")`},
		{`cd sub; pwd; ls -la`, `shell.Run("cd", "sub"); shell.Run("pwd"); shell.Run("ls", "-la")`},
		{`cd sub; [mkdir sub]; ls -la`, `shell.Run("cd", "sub"); shell.ExecErrOK("mkdir", "sub"); shell.Run("ls", "-la")`},
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
		{"cosh -i", `shell.Run("cosh", "-i")`},
		{"./cosh -i", `shell.Run("./cosh", "-i")`},
		{"cat main.go", `shell.Run("cat", "main.go")`},
		{"cd cogent", `shell.Run("cd", "cogent")`},
		{"cd cogent/", `shell.Run("cd", "cogent/")`},
		{"echo $PATH", `shell.Run("echo", "$PATH")`},
		{`"./Cogent Code"`, `shell.Run("./Cogent Code")`},
		{`./"Cogent Code"`, `shell.Run("./\"Cogent Code\"")`},
		{`Cogent\ Code`, `shell.Run("Cogent Code")`},
		{`./Cogent\ Code`, `shell.Run("./Cogent Code")`},
		{`ios\ deploy -i`, `shell.Run("ios deploy", "-i")`},
		{"./ios-deploy -i", `shell.Run("./ios-deploy", "-i")`},
		{"ios_deploy -i tree_file", `shell.Run("ios_deploy", "-i", "tree_file")`},
		{"ios_deploy/sub -i tree_file", `shell.Run("ios_deploy/sub", "-i", "tree_file")`},
		{"C:/ios_deploy/sub -i tree_file", `shell.Run("C:/ios_deploy/sub", "-i", "tree_file")`},
		{"ios_deploy -i tree_file/path", `shell.Run("ios_deploy", "-i", "tree_file/path")`},
		{"ios-deploy -i", `shell.Run("ios-deploy", "-i")`},
		{"ios-deploy -i tree-file", `shell.Run("ios-deploy", "-i", "tree-file")`},
		{"ios-deploy -i tree-file/path/here", `shell.Run("ios-deploy", "-i", "tree-file/path/here")`},
		{"cd ..", `shell.Run("cd", "..")`},
		{"cd ../another/dir/to/go_to", `shell.Run("cd", "../another/dir/to/go_to")`},
		{"cd ../an-other/dir/", `shell.Run("cd", "../an-other/dir/")`},
		{"curl https://google.com/search?q=hello%20world#body", `shell.Run("curl", "https://google.com/search?q=hello%20world#body")`},
	}
	sh := NewShell()
	for _, test := range tests {
		o := sh.TranspileLine(test.i)
		assert.Equal(t, test.e, o)
	}
}
