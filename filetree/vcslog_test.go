// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVCSLogWidget tests the log of a file: choosing revisions A and B by
// selecting commits and in its context menu, copying a revision id, and
// viewing the file at a revision.
func TestVCSLogWidget(t *testing.T) {
	defer core.TrackWindows()()
	dir, git := newGitRepo(t)
	fn := filepath.Join(dir, "a.go")
	require.NoError(t, os.WriteFile(fn, []byte("package a\n\nvar A = 2\n"), 0o644))
	git("commit", "-q", "-am", "second")
	ft := newVCSTree(t, dir)
	ft.AsyncLock()
	repo, _ := ft.FirstVCS()
	ft.AsyncUnlock()
	require.NotNil(t, repo)
	lg, err := repo.Log(fn, "")
	require.NoError(t, err)
	require.Len(t, lg, 2)

	b := core.NewBody()
	lv := NewVCSLog(b).SetRepo(repo).SetLog(lg).SetFile(fn)
	nw := core.AllRenderWindows.Len()
	core.RenderImage(b, core.Locked(b, func() {
		assert.Equal(t, "HEAD", lv.revisionA)
		tb := lv.ChildByName("log").(*core.Table)
		tb.SelectIndexEvent(1, events.SelectOne)
		assert.Equal(t, lg[1].Rev, lv.revisionA, "the first selection sets A")
		assert.False(t, lv.setA, "then B")
		tb.SelectIndexEvent(0, events.SelectOne)
		assert.Equal(t, lg[0].Rev, lv.revisionB)
		assert.True(t, lv.setA)

		menu := func() *core.Scene {
			m := core.NewScene()
			for _, cm := range tb.ContextMenus { // (the table's own, then the log's)
				cm(m, image.Point{})
			}
			return m
		}
		tb.SelectedIndex = 0
		core.ButtonByText(menu(), "Set Revision A").Send(events.Click)
		assert.Equal(t, lg[0].Rev, lv.revisionA)
		tb.SelectedIndex = 1
		core.ButtonByText(menu(), "Set Revision B").Send(events.Click)
		assert.Equal(t, lg[1].Rev, lv.revisionB)
		core.ButtonByText(menu(), "Copy Revision ID").Send(events.Click)
		md := lv.Clipboard().Read([]string{fileinfo.TextPlain})
		assert.Equal(t, lg[1].Rev, md.Text(fileinfo.TextPlain))

		core.ButtonByText(lv, "View A").Send(events.Click)
		core.ButtonByText(menu(), "View Revision").Send(events.Click)
	}))
	assert.Eventually(t, func() bool { return core.AllRenderWindows.Len() >= nw+2 }, 5*time.Second, 10*time.Millisecond, "the windows of the file at the revisions")
}

// TestFileInfoDiffWindows tests the windows of the file info of the
// selected files and of their diffs with the repository.
func TestFileInfoDiffWindows(t *testing.T) {
	defer core.TrackWindows()()
	dir, _ := newGitRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\n\nvar A = 3\n"), 0o644))
	ft := newVCSTree(t, dir)
	a := vcsNode(t, ft, dir, "a.go")
	nw := core.AllRenderWindows.Len()
	ft.AsyncLock()
	a.SelectEvent(events.SelectOne)
	a.showFileInfo()
	a.diffVCSSelected("", "")
	ft.AsyncUnlock()
	assert.Eventually(t, func() bool { return core.AllRenderWindows.Len() >= nw+2 }, 5*time.Second, 10*time.Millisecond, "the info and diff windows")
}
