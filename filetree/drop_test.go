// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"os"
	"path/filepath"
	"testing"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles/states"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dropAt sends an external drag event of the given type
// at the center of the row of the given node.
func (wt *watchTest) dropAt(typ events.Types, rel string, md mimedata.Mimes) {
	wt.b.AsyncLock()
	n := wt.node(rel)
	require.NotNil(wt.t, n, rel)
	bb := n.Parts.Geom.TotalBBox
	pos := bb.Min.Add(bb.Size().Div(2))
	sw := wt.ft.Events().RenderWindow().SystemWindow
	wt.b.AsyncUnlock()
	sw.Events().DragExternal(typ, pos, 0, md, false)
}

// TestExternalDrop tests dragging a file from another app over the
// tree, which shows the folder under the mouse as drag hovered,
// and dropping it on a folder, which copies it there.
func TestExternalDrop(t *testing.T) {
	wt := newWatchTest(t)
	src := filepath.Join(t.TempDir(), "new.txt")
	require.NoError(t, os.WriteFile(src, []byte("new\n"), 0o644))

	wt.dropAt(events.DragMove, "sub", nil)
	wt.waitFor("drag hovered", func() bool {
		return wt.hovered("sub")
	})
	wt.dropAt(events.DragMove, "a.go", nil)
	wt.waitFor("drag hover moved", func() bool {
		return wt.hovered("a.go") && !wt.hovered("sub")
	})

	wt.dropAt(events.Drop, "sub", mimedata.Mimes{mimedata.NewTextData(src)})
	wt.waitFor("copied", func() bool {
		b, err := os.ReadFile(wt.path("sub/new.txt"))
		return err == nil && string(b) == "new\n"
	})
	wt.waitFor("node", func() bool { return wt.hasNode("sub/new.txt") })
	assert.False(t, wt.hovered("a.go"))
	assert.False(t, wt.hovered("sub"))
}

// hovered returns whether the node for the relative path is drag hovered.
func (wt *watchTest) hovered(rel string) bool {
	wt.b.AsyncLock()
	defer wt.b.AsyncUnlock()
	return wt.node(rel).StateIs(states.DragHovered)
}
