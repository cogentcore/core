// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package transpile

import (
	"testing"

	_ "cogentcore.org/core/tensor/stats/metric"
	_ "cogentcore.org/core/tensor/stats/stats"
	_ "cogentcore.org/core/tensor/tmath"
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
		{"./goal -i", `./goal`},
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
	for _, test := range tests {
		toks := TokensFromString(test.i)
		p, _ := toks.Path(false)
		assert.Equal(t, test.e, p)
	}
}

// these are more general tests of full-line statements of various forms
func TestTranspile(t *testing.T) {
	// logx.UserLevel = slog.LevelDebug
	tests := []exIn{
		{"ls", `goal.Run("ls")`},
		{"$ls -la$", `goal.Run("ls", "-la")`},
		{"ls -la", `goal.Run("ls", "-la")`},
		{"chmod +x file", `goal.Run("chmod", "+x", "file")`},
		{"ls --help", `goal.Run("ls", "--help")`},
		{"ls go", `goal.Run("ls", "go")`},
		{"cd go", `goal.Run("cd", "go")`},
		{`var name string`, `var name string`},
		{`name = "test"`, `name = "test"`},
		{`echo {name}`, `goal.Run("echo", name)`},
		{`echo "testing"`, `goal.Run("echo", "testing")`},
		{`number := 1.23`, `number := 1.23`},
		{`res1, res2 := FunTwoRet()`, `res1, res2 := FunTwoRet()`},
		{`res1, res2, res3 := FunThreeRet()`, `res1, res2, res3 := FunThreeRet()`},
		{`println("hi")`, `println("hi")`},
		{`fmt.Println("hi")`, `fmt.Println("hi")`},
		{`for i := 0; i < 3; i++ { fmt.Println(i, "\n")`, `for i := 0; i < 3; i++ { fmt.Println(i, "\n")`},
		{"for i, v := range $ls -la$ {", `for i, v := range goal.Output("ls", "-la") {`},
		{`// todo: fixit`, `// todo: fixit`},
		{"$go build$", `goal.Run("go", "build")`},
		{"{go build()}", `go build()`},
		{"go build", `goal.Run("go", "build")`},
		{"go build()", `go build()`},
		{"go build &", `goal.Start("go", "build")`},
		{"[mkdir subdir]", `goal.RunErrOK("mkdir", "subdir")`},
		{"set something hello-1", `goal.Run("set", "something", "hello-1")`},
		{"set something = hello", `goal.Run("set", "something", "=", "hello")`},
		{`set something = "hello"`, `goal.Run("set", "something", "=", "hello")`},
		{`set something=hello`, `goal.Run("set", "something=hello")`},
		{`set "something=hello"`, `goal.Run("set", "something=hello")`},
		{`set something="hello"`, `goal.Run("set", "something=\"hello\"")`},
		{`add-path /opt/sbin /opt/homebrew/bin`, `goal.Run("add-path", "/opt/sbin", "/opt/homebrew/bin")`},
		{`cat file > test.out`, `goal.Run("cat", "file", ">", "test.out")`},
		{`cat file | grep -v exe > test.out`, `goal.Start("cat", "file", "|"); goal.Run("grep", "-v", "exe", ">", "test.out")`},
		{`cd sub; pwd; ls -la`, `goal.Run("cd", "sub"); goal.Run("pwd"); goal.Run("ls", "-la")`},
		{`cd sub; [mkdir sub]; ls -la`, `goal.Run("cd", "sub"); goal.RunErrOK("mkdir", "sub"); goal.Run("ls", "-la")`},
		{`cd sub; mkdir names[4]`, `goal.Run("cd", "sub"); goal.Run("mkdir", "names[4]")`},
		{"ls -la > test.out", `goal.Run("ls", "-la", ">", "test.out")`},
		{"ls > test.out", `goal.Run("ls", ">", "test.out")`},
		{"ls -la >test.out", `goal.Run("ls", "-la", ">", "test.out")`},
		{"ls -la >> test.out", `goal.Run("ls", "-la", ">>", "test.out")`},
		{"ls -la >& test.out", `goal.Run("ls", "-la", ">&", "test.out")`},
		{"ls -la >>& test.out", `goal.Run("ls", "-la", ">>&", "test.out")`},
		{"@1 ls -la", `goal.Run("@1", "ls", "-la")`},
		{"git switch main", `goal.Run("git", "switch", "main")`},
		{"git checkout 123abc", `goal.Run("git", "checkout", "123abc")`},
		{"go get cogentcore.org/core@main", `goal.Run("go", "get", "cogentcore.org/core@main")`},
		{"ls *.go", `goal.Run("ls", "*.go")`},
		{"ls ??.go", `goal.Run("ls", "??.go")`},
		{`fmt.Println("hi")`, `fmt.Println("hi")`},
		{"goal -i", `goal.Run("goal", "-i")`},
		{"./goal -i", `goal.Run("./goal", "-i")`},
		{"cat main.go", `goal.Run("cat", "main.go")`},
		{"cd cogent", `goal.Run("cd", "cogent")`},
		{"cd cogent/", `goal.Run("cd", "cogent/")`},
		{"echo $PATH", `goal.Run("echo", "$PATH")`},
		{`"./Cogent Code"`, `goal.Run("./Cogent Code")`},
		{`./"Cogent Code"`, `goal.Run("./\"Cogent Code\"")`},
		{`Cogent\ Code`, `goal.Run("Cogent Code")`},
		{`./Cogent\ Code`, `goal.Run("./Cogent Code")`},
		{`ios\ deploy -i`, `goal.Run("ios deploy", "-i")`},
		{"./ios-deploy -i", `goal.Run("./ios-deploy", "-i")`},
		{"ios_deploy -i tree_file", `goal.Run("ios_deploy", "-i", "tree_file")`},
		{"ios_deploy/sub -i tree_file", `goal.Run("ios_deploy/sub", "-i", "tree_file")`},
		{"C:/ios_deploy/sub -i tree_file", `goal.Run("C:/ios_deploy/sub", "-i", "tree_file")`},
		{"ios_deploy -i tree_file/path", `goal.Run("ios_deploy", "-i", "tree_file/path")`},
		{"ios-deploy -i", `goal.Run("ios-deploy", "-i")`},
		{"ios-deploy -i tree-file", `goal.Run("ios-deploy", "-i", "tree-file")`},
		{"ios-deploy -i tree-file/path/here", `goal.Run("ios-deploy", "-i", "tree-file/path/here")`},
		{"cd ..", `goal.Run("cd", "..")`},
		{"cd ../another/dir/to/go_to", `goal.Run("cd", "../another/dir/to/go_to")`},
		{"cd ../an-other/dir/", `goal.Run("cd", "../an-other/dir/")`},
		{"curl https://google.com/search?q=hello%20world#body", `goal.Run("curl", "https://google.com/search?q=hello%20world#body")`},
		{"func splitLines(str string) []string {", `splitLines := func(str string)[]string {`},
		{"type Result struct {", `type Result struct {`},
		{"var Jobs *table.Table", `var Jobs *table.Table`},
		{"type Result struct { JobID string", `type Result struct { JobID string`},
		{"type Result struct { JobID string `width:\"60\"`", "type Result struct { JobID string `width:\"60\"`"},
		{"func RunInExamples(fun func()) {", "RunInExamples := func(fun func()) {"},
		{"ctr++", "ctr++"},
		{"ctr--", "ctr--"},
		{"stru.ctr++", "stru.ctr++"},
		{"stru.ctr--", "stru.ctr--"},
		{"meta += ln", "meta += ln"},
		{"var data map[string]any", "var data map[string]any"},
		// non-math-mode tensor indexing:
		{"x = a[1,f(2,3)]", `x = a.Value(int(1), int(f(2, 3)))`},
		{"x = a[1]", `x = a[1]`},
		{"x = a[f(2,3)]", `x = a[f(2, 3)]`},
		{"a[1,2] = 55", `a.Set(55, int(1), int(2))`},
		{"a[1,2] = 55 // and that is good", `a.Set(55, int(1), int(2)) // and that is good`},
		{"a[1,2] += f(2,55)", `a.SetAdd(f(2, 55), int(1), int(2))`},
		{"a[1,2] *= f(2,55)", `a.SetMul(f(2, 55), int(1), int(2))`},
		{"Data[idx, Integ] = integ", `Data.Set(integ, int(idx), int(Integ))`},
		{"Data[Idxs[idx, 25], Integ] = integ", `Data.Set(integ, int(Idxs.Value(int(idx), int(25))), int(Integ))`},
		{"Layers[NeuronIxs[NrnLayIndex, ni]].GatherSpikes(&Ctx[0], ni, di)", `Layers[NeuronIxs.Value(int(NrnLayIndex), int(ni))].GatherSpikes( & Ctx[0], ni, di)`},
	}

	st := NewState()
	for _, test := range tests {
		o := st.TranspileLine(test.i)
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
			`goal.AddCommand("list", func(args ...string) {
goal.Run("ls", "-la", "args...")
})`},
	}

	st := NewState()
	for _, test := range tests {
		st.TranspileCode(test.i)
		o := st.Code()
		assert.Equal(t, test.e, o)
	}
}

// Use this for testing the current thing working on.
func TestCur(t *testing.T) {
	// logx.UserLevel = slog.LevelDebug
	tests := []exIn{
		{"Layers[NeuronIxs[NrnLayIndex, ni]].GatherSpikes(&Ctx[0], ni, di)", `Layers[NeuronIxs.Value(int(NrnLayIndex), int(ni))].GatherSpikes( & Ctx[0], ni, di)`},
	}
	st := NewState()
	st.MathRecord = false
	for _, test := range tests {
		o := st.TranspileLine(test.i)
		assert.Equal(t, test.e, o)
	}
}

func TestMath(t *testing.T) {
	// logx.UserLevel = slog.LevelDebug
	tests := []exIn{
		{"# x := 1", `x := tensor.Tensor(tensor.NewIntScalar(1))`},
		{"# x := a + 1", `x := tensor.Tensor(tmath.Add(a, tensor.NewIntScalar(1)))`},
		{"# x = x * 4", `x = tmath.Mul(x, tensor.NewIntScalar(4))`},
		{"# a = x + y", `a = tmath.Add(x, y)`},
		{"# a := x ** 2", `a := tensor.Tensor(tmath.Pow(x, tensor.NewIntScalar(2)))`},
		{"# a = -x", `a = tmath.Negate(x)`},
		{"# x @ a", `matrix.Mul(x, a)`},
		{"# x += 1", `tmath.AddAssign(x, tensor.NewIntScalar(1))`},
		{"# a := [1,2,3,4]", `a := tensor.Tensor(tensor.NewIntFromValues([]int { 1, 2, 3, 4 }  ...))`},
		{"# a := [1.,2,3,4]", `a := tensor.Tensor(tensor.NewFloat64FromValues([]float64 { 1., 2, 3, 4 }  ...))`},
		{"# a := [[1,2], [3,4]]", `a := tensor.Tensor(tensor.Reshape(tensor.NewIntFromValues([]int { 1, 2, 3, 4 }  ...), 2, 2))`},
		{"# a.ndim", `tensor.NewIntScalar(a.NumDims())`},
		{"# ndim(a)", `tensor.NewIntScalar(a.NumDims())`},
		{"# x.shape", `tensor.NewIntFromValues(x.Shape().Sizes ...)`},
		{"# x.T", `tensor.Transpose(x)`},
		{"# zeros(3, 4)", `tensor.NewFloat64(3, 4)`},
		{"# full(5.5, 3, 4)", `tensor.NewFloat64Full(5.5, 3, 4)`},
		{"# zeros(sh)", `tensor.NewFloat64(tensor.AsIntSlice(sh) ...)`},
		{"# arange(36)", `tensor.NewIntRange(36)`},
		{"# arange(36, 0, -1)", `tensor.NewIntRange(36, 0,  - 1)`},
		{"# linspace(0, 5, 6, true)", `tensor.NewFloat64SpacedLinear(tensor.NewIntScalar(0), tensor.NewIntScalar(5), 6, true)`},
		{"# reshape(x, 6, 6)", `tensor.Reshape(x, 6, 6)`},
		{"# reshape(x, [6, 6])", `tensor.Reshape(x, 6, 6)`},
		{"# reshape(x, sh)", `tensor.Reshape(x, tensor.AsIntSlice(sh) ...)`},
		{"# reshape(arange(36), 6, 6)", `tensor.Reshape(tensor.NewIntRange(36), 6, 6)`},
		{"# a.reshape(6, 6)", `tensor.Reshape(a, 6, 6)`},
		{"# a[1, 2]", `tensor.Reslice(a, 1, 2)`},
		{"# a[:, 2]", `tensor.Reslice(a, tensor.FullAxis, 2)`},
		{"# a[1:3:1, 2]", `tensor.Reslice(a, tensor.Slice { Start:1, Stop:3, Step:1 } , 2)`},
		{"# a[::-1, 2]", `tensor.Reslice(a, tensor.Slice { Step: - 1 } , 2)`},
		{"# a[:3, 2]", `tensor.Reslice(a, tensor.Slice { Stop:3 } , 2)`},
		{"# a[2:, 2]", `tensor.Reslice(a, tensor.Slice { Start:2 } , 2)`},
		{"# a[2:, 2, newaxis]", `tensor.Reslice(a, tensor.Slice { Start:2 } , 2, tensor.NewAxis)`},
		{"# a[..., 2:]", `tensor.Reslice(a, tensor.Ellipsis, tensor.Slice { Start:2 } )`},
		{"# a[:, 2] = b", `tmath.Assign(tensor.Reslice(a, tensor.FullAxis, 2), b)`},
		{"# a[:, 2] += b", `tmath.AddAssign(tensor.Reslice(a, tensor.FullAxis, 2), b)`},
		{"# cos(a)", `tmath.Cos(a)`},
		{"# stats.Mean(a)", `stats.Mean(a)`},
		{"# (stats.Mean(a))", `(stats.Mean(a))`},
		{"# stats.Mean(reshape(a,36))", `stats.Mean(tensor.Reshape(a, 36))`},
		{"# z = a[1:5,1:5] - stats.Mean(ra)", `z = tmath.Sub(tensor.Reslice(a, tensor.Slice { Start:1, Stop:5 } , tensor.Slice { Start:1, Stop:5 } ), stats.Mean(ra))`},
		{"# metric.Matrix(metric.Cosine, a)", `metric.Matrix(metric.Cosine, a)`},
		{"# a > 5", `tmath.Greater(a, tensor.NewIntScalar(5))`},
		{"# !a", `tmath.Not(a)`},
		{"# a[a > 5]", `tensor.Mask(a, tmath.Greater(a, tensor.NewIntScalar(5)))`},
		{"# a[a > 5].flatten()", `tensor.Flatten(tensor.Mask(a, tmath.Greater(a, tensor.NewIntScalar(5))))`},
		{"# a[:3, 2].copy()", `tensor.Clone(tensor.Reslice(a, tensor.Slice { Stop:3 } , 2))`},
		{"# a[:3, 2].reshape(4,2)", `tensor.Reshape(tensor.Reslice(a, tensor.Slice { Stop:3 } , 2), 4, 2)`},
		{"# a > 5 || a < 1", `tmath.Or(tmath.Greater(a, tensor.NewIntScalar(5)), tmath.Less(a, tensor.NewIntScalar(1)))`},
		{"# fmt.Println(a)", `fmt.Println(a)`},
		{"# }", `}`},
		{"# if a[1,2] == 2 {", `if tmath.Equal(tensor.Reslice(a, 1, 2), tensor.NewIntScalar(2)).Bool1D(0) {`},
		{"# for i := 0; i < 3; i++ {", `for i := tensor.Tensor(tensor.NewIntScalar(0)); tmath.Less(i, tensor.NewIntScalar(3)).Bool1D(0); tmath.Inc(i) {`},
		{"# for i, v := range a {", `for i := 0; i < a.Len(); i++ { v := a .Float1D(i)`},
		{`# x := get("item")`, `x := tensor.Tensor(datafs.Get("item"))`},
		{`# set("item", x)`, `datafs.Set("item", x)`},
		{`# set("item", 5)`, `datafs.Set("item", tensor.NewIntScalar(5))`},
		{`fmt.Println(#zeros(3,4)#)`, `fmt.Println(tensor.NewFloat64(3, 4))`},
	}

	st := NewState()
	st.MathRecord = false
	for _, test := range tests {
		o := st.TranspileLine(test.i)
		assert.Equal(t, test.e, o)
	}
}
