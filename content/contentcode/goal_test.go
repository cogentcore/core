// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package contentcode

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// goalPageMD is a page with Goal code blocks, in [TestGoalPages].
const goalPageMD = "# Goal\n\n```Goal\nx := 1\n```\n\nText.\n\n```Goal\nfmt.Println(x)\n```\n\n```Goal\nx = \"a\"\n```\n"

// newGoalApp returns a copy of the test app, in the module, with a page of
// Goal code.
func newGoalApp(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("testdata", "tmp-goal")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	require.NoError(t, os.CopyFS(dir, os.DirFS("testdata/app")))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "content", "goal.md"), []byte(goalPageMD), 0o666))
	return dir
}

// goalPaths returns the import paths of the Goal code, by package name.
func goalPaths() map[string]string {
	paths := map[string]string{}
	for _, p := range append(goalImports, defaultImports...) {
		paths[pathName(p)] = p
	}
	return paths
}

// TestGoalPages tests the Go code generated for the Goal code blocks of a
// page, given their transpiled code, as Lab's transpiler makes it: the
// blocks that do not parse or type check are left out, with those after
// them, and the unused variables are used.
func TestGoalPages(t *testing.T) {
	dir := newGoalApp(t)
	pages, err := readGoalPages(dir, "content")
	require.NoError(t, err)
	require.Len(t, pages, 1)
	pg := pages[0]
	assert.Equal(t, "goal.md", pg.key)
	require.Len(t, pg.blocks, 3)
	assert.Equal(t, "x := 1", strings.TrimSpace(pg.blocks[0].src))
	assert.Equal(t, 4, pg.blocks[0].line)

	// as transpiled
	pg.blocks[0].gocode = "x := 1\ny := 2"
	pg.blocks[1].gocode = "fmt.Println(x)\ncore.NewText(b).SetText(\"x\")"
	pg.blocks[2].gocode = "x = \"a\""
	pg.name = goalPrefix + "0"
	paths := goalPaths()
	pg.prepare(paths, false)
	assert.Equal(t, 3, pg.n, "all of them parse")
	assert.Contains(t, pg.code, "func contentGoal0() content.GoalBlock")
	assert.Contains(t, pg.code, "fmt.Fprintln(")
	assert.True(t, pg.pkgs["fmt"])
	assert.Equal(t, 1, pg.blockAtLine(pg.start+1+strings.Count(pg.code[:strings.Index(pg.code, "contentcode:block 1")], "\n")))
	assert.Equal(t, 0, pg.blockAtLine(pg.start+1))

	pkgName, err := packageName(dir, "")
	require.NoError(t, err)
	src, err := typeCheckBlocks(dir, filepath.Join(dir, "content_gen.go"), pkgName, "hash", "", nil, pages, paths, false)
	require.NoError(t, err)
	assert.Equal(t, 2, pg.n, "the block that does not type check is left out")
	assert.Contains(t, pg.blocks[2].err, "cannot use")
	out := string(src)
	assert.Contains(t, out, "content.AddCompiledGoal(")
	assert.Contains(t, out, `"goal.md"`)
	assert.Contains(t, out, "_ = y", "the unused variable")
	assert.Contains(t, out, "Goal code from here on (1 blocks)")

	// recording Goal math mode uses tensorfs
	pg.n = 1
	require.NoError(t, pg.generate(paths, true))
	assert.Contains(t, pg.code, "tensorfs.ListOutput")
	assert.True(t, pg.pkgs[tensorfsPath])

	// a block that does not parse, and those after it, are left out
	pg.blocks[1].gocode = "x := ("
	pg.blocks[1].err = ""
	pg.prepare(paths, false)
	assert.Equal(t, 1, pg.n)
	assert.NotEmpty(t, pg.blocks[1].err)
	pg.truncate(0, "a reason", paths, false)
	assert.Equal(t, 0, pg.n)
	assert.Equal(t, "", pg.code)
	assert.Equal(t, "a reason", pg.blocks[0].err)
}

// TestGoalNoLab tests that the Goal code blocks of an app whose module
// does not require Lab are not compiled, while its Go code blocks are.
func TestGoalNoLab(t *testing.T) {
	dir := newGoalApp(t)
	pages, err := readGoalPages(dir, "content")
	require.NoError(t, err)
	err = transpileGoal(dir, pages, "", false)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errNoLab), "%v", err)
	assert.NoError(t, transpileGoal(dir, nil, "", false), "no pages")

	c, err := App(dir)
	require.NoError(t, err)
	ok, err := Generate(c)
	require.NoError(t, err)
	assert.True(t, ok)
	out, err := os.ReadFile(filepath.Join(dir, c.Output))
	require.NoError(t, err)
	assert.Contains(t, string(out), "contentCode0")
	assert.NotContains(t, string(out), "AddCompiledGoal")
}

// TestParseArgsImports tests the repeatable flags of the arguments, and
// the helpers of the messages.
func TestParseArgsImports(t *testing.T) {
	c, err := ParseArgs("contentcode", []string{"-import", "a/b", "-import", "x=c/d", "-interpret", "page", "-force"})
	require.NoError(t, err)
	assert.Equal(t, []string{"a/b", "x=c/d"}, c.Imports)
	assert.Equal(t, []string{"page"}, c.Interpret)
	assert.True(t, c.Force)
	f := importFlags(c.Imports)
	assert.Equal(t, "a/b,x=c/d", f.String())
	_, err = ParseArgs("contentcode", []string{"-nosuch"})
	assert.Error(t, err)

	assert.Equal(t, 3, abs(-3))
	assert.Equal(t, 3, abs(3))
	assert.Equal(t, "3: msg", typeError{line: 3, msg: "msg"}.String())
	assert.Equal(t, filepath.Join("content", "a.md"), displayPath(".", "content/a.md"))
	assert.True(t, filepath.IsAbs(displayPath(os.TempDir(), "a.md")), "outside of the current directory")
}
