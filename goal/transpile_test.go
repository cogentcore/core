// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type exIn struct {
	i string
	e string
}

type wexIn struct {
	i     string
	isErr bool
	e     []string
}

// these are more general tests of full-line statements of various forms
func TestExecWords(t *testing.T) {
	tests := []wexIn{
		{`ls`, false, []string{`ls`}},
		{`cat "be"`, false, []string{`cat`, `"be"`}},
		{`cat "be`, true, []string{`cat`, `"be`}},
		{`cat "be a thing"`, false, []string{`cat`, `"be a thing"`}},
		{`cat "{be \"a\" thing}"`, false, []string{`cat`, `"{be \"a\" thing}"`}},
		{`cat {vals[1:10]}`, false, []string{`cat`, `{`, `vals[1:10]`, `}`}},
		{`cat {myfunc(vals[1:10], "test", false)}`, false, []string{`cat`, `{`, `myfunc(vals[1:10],"test",false)`, `}`}},
		{`cat vals[1:10]`, false, []string{`cat`, `vals[1:10]`}},
		{`cat vals...`, false, []string{`cat`, `vals...`}},
		{`[cat vals...]`, false, []string{`[`, `cat`, `vals...`, `]`}},
		{`[cat vals...]; ls *.tsv`, false, []string{`[`, `cat`, `vals...`, `]`, `;`, `ls`, `*.tsv`}},
		{`cat vals... | grep -v "b"`, false, []string{`cat`, `vals...`, `|`, `grep`, `-v`, `"b"`}},
		{`cat vals...>&file.out`, false, []string{`cat`, `vals...`, `>&`, `file.out`}},
		{`cat vals...>&@0:file.out`, false, []string{`cat`, `vals...`, `>&`, `@0:file.out`}},
		{`./"Cogent Code"`, false, []string{`./"Cogent Code"`}},
		{`Cogent\ Code`, false, []string{`Cogent Code`}},
		{`./Cogent\ Code`, false, []string{`./Cogent Code`}},
	}
	for _, test := range tests {
		o, err := ExecWords(test.i)
		assert.Equal(t, test.e, o)
		if err != nil {
			if !test.isErr {
				t.Error("should not have been an error:", test.i)
			}
		} else if test.isErr {
			t.Error("was supposed to be an error:", test.i)
		}
	}
}

// Paths tests the Path() code
func TestPaths(t *testing.T) {
	// logx.UserLevel = slog.LevelDebug
	tests := []exIn{
		{`fmt.Println("hi")`, `fmt.Println`},
		{"./cosh -i", `./cosh`},
		{"main.go", `main.go`},
		{"cogent/", `cogent/`},
		{`./"Cogent Code"`, `./\"Cogent Code\"`},
		{`Cogent\ Code`, ``},
		{`./Cogent\ Code`, `./Cogent Code`},
		{"./ios-deploy", `./ios-deploy`},
		{"ios_deploy/sub", `ios_deploy/sub`},
		{"C:/ios_deploy/sub", `C:/ios_deploy/sub`},
		{"..", `..`},
		{"../another/dir/to/go_to", `../another/dir/to/go_to`},
		{"../an-other/dir/", `../an-other/dir/`},
		{"https://google.com/search?q=hello%20world#body", `https://google.com/search?q=hello%20world#body`},
	}
	gl := NewGoal()
	for _, test := range tests {
		toks := gl.Tokens(test.i)
		p, _ := toks.Path(false)
		assert.Equal(t, test.e, p)
	}
}

// these are more general tests of full-line statements of various forms
func TestTranspile(t *testing.T) {
	// logx.UserLevel = slog.LevelDebug
	tests := []exIn{
		{"ls", `shell.Run("ls")`},
		{"`ls -la`", `shell.Run("ls", "-la")`},
		{"ls -la", `shell.Run("ls", "-la")`},
		{"ls --help", `shell.Run("ls", "--help")`},
		{"ls go", `shell.Run("ls", "go")`},
		{"cd go", `shell.Run("cd", "go")`},
		{`var name string`, `var name string`},
		{`name = "test"`, `name = "test"`},
		{`echo {name}`, `shell.Run("echo", name)`},
		{`echo "testing"`, `shell.Run("echo", "testing")`},
		{`number := 1.23`, `number := 1.23`},
		{`res1, res2 := FunTwoRet()`, `res1, res2 := FunTwoRet()`},
		{`res1, res2, res3 := FunThreeRet()`, `res1, res2, res3 := FunThreeRet()`},
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
		{"[mkdir subdir]", `shell.RunErrOK("mkdir", "subdir")`},
		{"set something hello-1", `shell.Run("set", "something", "hello-1")`},
		{"set something = hello", `shell.Run("set", "something", "=", "hello")`},
		{`set something = "hello"`, `shell.Run("set", "something", "=", "hello")`},
		{`set something=hello`, `shell.Run("set", "something=hello")`},
		{`set "something=hello"`, `shell.Run("set", "something=hello")`},
		{`set something="hello"`, `shell.Run("set", "something=\"hello\"")`},
		{`add-path /opt/sbin /opt/homebrew/bin`, `shell.Run("add-path", "/opt/sbin", "/opt/homebrew/bin")`},
		{`cat file > test.out`, `shell.Run("cat", "file", ">", "test.out")`},
		{`cat file | grep -v exe > test.out`, `shell.Start("cat", "file", "|"); shell.Run("grep", "-v", "exe", ">", "test.out")`},
		{`cd sub; pwd; ls -la`, `shell.Run("cd", "sub"); shell.Run("pwd"); shell.Run("ls", "-la")`},
		{`cd sub; [mkdir sub]; ls -la`, `shell.Run("cd", "sub"); shell.RunErrOK("mkdir", "sub"); shell.Run("ls", "-la")`},
		{`cd sub; mkdir names[4]`, `shell.Run("cd", "sub"); shell.Run("mkdir", "names[4]")`},
		{"ls -la > test.out", `shell.Run("ls", "-la", ">", "test.out")`},
		{"ls > test.out", `shell.Run("ls", ">", "test.out")`},
		{"ls -la >test.out", `shell.Run("ls", "-la", ">", "test.out")`},
		{"ls -la >> test.out", `shell.Run("ls", "-la", ">>", "test.out")`},
		{"ls -la >& test.out", `shell.Run("ls", "-la", ">&", "test.out")`},
		{"ls -la >>& test.out", `shell.Run("ls", "-la", ">>&", "test.out")`},
		{"@1 ls -la", `shell.Run("@1", "ls", "-la")`},
		{"git switch main", `shell.Run("git", "switch", "main")`},
		{"git checkout 123abc", `shell.Run("git", "checkout", "123abc")`},
		{"go get cogentcore.org/core@main", `shell.Run("go", "get", "cogentcore.org/core@main")`},
		{"ls *.go", `shell.Run("ls", "*.go")`},
		{"ls ??.go", `shell.Run("ls", "??.go")`},
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
		{"func splitLines(str string) []string {", `splitLines := func(str string)[]string {`},
		{"type Result struct {", `type Result struct {`},
		{"var Jobs *table.Table", `var Jobs *table.Table`},
		{"type Result struct { JobID string", `type Result struct { JobID string`},
		{"type Result struct { JobID string `width:\"60\"`", "type Result struct { JobID string `width:\"60\"`"},
		{"func RunInExamples(fun func()) {", "RunInExamples := func(fun func()) {"},
		{"ctr++", "ctr++"},
		{"stru.ctr++", "stru.ctr++"},
		{"meta += ln", "meta += ln"},
		{"var data map[string]any", "var data map[string]any"},
	}

	gl := NewGoal()
	for _, test := range tests {
		o := gl.TranspileLine(test.i)
		assert.Equal(t, test.e, o)
	}
}

// tests command generation
func TestCommand(t *testing.T) {
	// logx.UserLevel = slog.LevelDebug
	tests := []exIn{
		{
			`command list {
ls -la args... 
}`,
			`shell.AddCommand("list", func(args ...string) {
shell.Run("ls", "-la", "args...")
})`},
	}

	gl := NewGoal()
	for _, test := range tests {
		gl.TranspileCode(test.i)
		o := gl.Code()
		assert.Equal(t, test.e, o)
	}
}

func TestMath(t *testing.T) {
	// logx.UserLevel = slog.LevelDebug
	tests := []exIn{
		{"$x := 1", `x := tensor.NewFloat64Scalar(1)`},
	}

	gl := NewGoal()
	for _, test := range tests {
		o := gl.TranspileLine(test.i)
		assert.Equal(t, test.e, o)
	}
}
