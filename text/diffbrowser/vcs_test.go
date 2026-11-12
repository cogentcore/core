// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package diffbrowser

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"cogentcore.org/core/base/vcs"
	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDiffVCSTree tests that the tree of the files changed between two
// revisions has each directory once, with its subdirectories and then its
// files, each sorted by name, as in a file tree, for files in nested
// directories, at the top level, and with spaces in their paths.
func TestDiffVCSTree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	git := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
		return strings.TrimSpace(string(out))
	}
	files := []string{"z.txt", "a/b.go", "a/c/d.go", "a/e.go", "a b/sp.txt", "b/x.go", "old.txt"}
	write := func(content string) {
		for _, f := range files {
			p := filepath.Join(dir, f)
			require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o755))
			require.NoError(t, os.WriteFile(p, []byte(f+content), 0o644))
		}
	}
	commit := func() string {
		git("add", "-A")
		git("-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-q", "-m", "c")
		return git("rev-parse", "HEAD")
	}
	git("init", "-q", "-b", "main")
	write("\n")
	revA := commit()
	files = files[:len(files)-1] // old.txt is deleted
	require.NoError(t, os.Remove(filepath.Join(dir, "old.txt")))
	files = append(files, "a/c/new.go") // and new.go is added
	write("\nchanged\n")
	revB := commit()

	repo, err := vcs.NewRepo("origin", dir)
	require.NoError(t, err)
	br := NewBrowser(core.NewBody())
	br.UpdateTree()
	br.DiffVCS(repo, revA, revB)

	var lines []string
	var walk func(n *Node, depth int)
	walk = func(n *Node, depth int) {
		for _, c := range n.Children {
			cn := c.(*Node)
			lines = append(lines, strings.Repeat("  ", depth)+cn.Text)
			walk(cn, depth+1)
		}
	}
	walk(br.Tree(), 0)
	assert.Equal(t, []string{
		"a",
		"  c",
		"    d.go [M]",
		"    new.go [A]",
		"  b.go [M]",
		"  e.go [M]",
		"a b",
		"  sp.txt [M]",
		"b",
		"  x.go [M]",
		"old.txt [D]",
		"z.txt [M]",
	}, lines)

	// the files of the nodes, with their contents at the revisions
	var dgo *Node
	br.Tree().WalkDown(func(n tree.Node) bool {
		if nn, ok := n.(*Node); ok && nn.Text == "d.go [M]" {
			dgo = nn
		}
		return tree.Continue
	})
	require.NotNil(t, dgo)
	assert.Equal(t, filepath.Join(dir, "a", "c", "d.go"), dgo.FileB)
	assert.Equal(t, "a/c/d.go\n", dgo.TextA)
	assert.Equal(t, "a/c/d.go\nchanged\n", dgo.TextB)
}

// TestParseChanges tests reading the changed files of git diff
// --name-status: renamed files (with a score) in the directories of their
// new paths, and paths with spaces.
func TestParseChanges(t *testing.T) {
	top := parseChanges([]string{
		"M\tdocs/a b.md",
		"R087\told/x.go\tnew/sub/x.go",
		"A\ttop.txt",
		"",
	})
	require.Len(t, top.files, 1)
	assert.Equal(t, fileChange{"A", "top.txt", "top.txt"}, top.files[0])
	require.Contains(t, top.dirs, "docs")
	assert.Equal(t, fileChange{"M", "docs/a b.md", "docs/a b.md"}, top.dirs["docs"].files[0])
	require.Contains(t, top.dirs, "new")
	sub := top.dirs["new"].dirs["sub"]
	require.NotNil(t, sub)
	assert.Equal(t, fileChange{"R", "old/x.go", "new/sub/x.go"}, sub.files[0])
	assert.NotContains(t, top.dirs, "old")
}
