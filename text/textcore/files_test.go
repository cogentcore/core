// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package textcore

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/base/vcs"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/text/lines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSave tests saving, which does not overwrite a file
// that has changed on disk without asking.
func TestSave(t *testing.T) {
	fname := filepath.Join(t.TempDir(), "f.txt")
	require.NoError(t, os.WriteFile(fname, []byte("one\n"), 0o644))
	ln := lines.NewLines()
	require.NoError(t, ln.Open(fname))
	assert.False(t, ln.ChangedOnDisk())

	b := core.NewBody()
	showed := make(chan struct{})
	b.OnFinal(events.Show, func(e events.Event) { showed <- struct{}{} })
	b.RunWindow()
	<-showed
	save := func() error {
		b.AsyncLock()
		defer b.AsyncUnlock()
		return Save(b.Scene, ln)
	}

	ln.SetText([]byte("two\n"))
	require.NoError(t, save())
	got, _ := os.ReadFile(fname)
	assert.Equal(t, "two\n", string(got))
	assert.False(t, ln.IsNotSaved())
	assert.False(t, ln.ChangedOnDisk(), "saving updates the time")

	// changed on disk: prompts, without saving
	time.Sleep(10 * time.Millisecond) // a different modification time
	require.NoError(t, os.WriteFile(fname, []byte("disk\n"), 0o644))
	assert.True(t, ln.ChangedOnDisk())
	ln.SetText([]byte("three\n"))
	require.NoError(t, save())
	got, _ = os.ReadFile(fname)
	assert.Equal(t, "disk\n", string(got))
}

// TestDiffEditorFromRevs tests that the diff of a file with no revisions
// given is of the last commit (HEAD) and the current file.
func TestDiffEditorFromRevs(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	dir, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	git := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, string(out))
	}
	fname := filepath.Join(dir, "f.txt")
	require.NoError(t, os.WriteFile(fname, []byte("one\n"), 0o644))
	git("init", "-q")
	git("add", ".")
	git("-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-q", "-m", "init")
	require.NoError(t, os.WriteFile(fname, []byte("two\n"), 0o644))
	repo, err := vcs.NewRepo("", dir)
	require.NoError(t, err)

	b := core.NewBody()
	showed := make(chan struct{})
	b.OnFinal(events.Show, func(e events.Event) { showed <- struct{}{} })
	b.RunWindow()
	<-showed
	b.AsyncLock()
	defer b.AsyncUnlock()
	de, err := DiffEditorDialogFromRevs(b, repo, fname, nil, "", "")
	require.NoError(t, err)
	assert.Equal(t, "HEAD", de.RevisionA)
	assert.Contains(t, de.linesA.String(), "one")
	assert.Contains(t, de.linesB.String(), "two")
}
