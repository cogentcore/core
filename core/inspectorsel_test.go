// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/base/profile"
	"cogentcore.org/core/events"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInspectorSelectElement tests selecting a widget of a window by
// clicking it in the selection mode of an inspector of the window, in
// which the boxes of the widgets are rendered, and the toolbar and the
// window of an inspector.
func TestInspectorSelectElement(t *testing.T) {
	b := NewBody()
	target := NewButton(b).SetText("Target")
	b.runAndShowNewWindow()
	defer func() {
		b.AsyncLock()
		b.Close()
		b.AsyncUnlock()
	}()

	ib := NewBody()
	makeInspector(ib, b.Scene)
	var is *Inspector
	RenderImage(ib, Locked(ib, func() {
		is = tree.ChildByType[*Inspector](ib)
		require.NotNil(t, ButtonByText(ib.Scene, "Select element"), "the toolbar")
		is.toggleSelectionMode()
	}), func() {
		// the selection mode is turned on in the window of the scene
		require.Eventually(t, func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return b.Scene.renderBBoxes
		}, 5*time.Second, 10*time.Millisecond)
		b.AsyncLock()
		b.Scene.NeedsRender() // with the boxes of the widgets
		bb := target.Geom.TotalBBox
		b.AsyncUnlock()
		time.Sleep(100 * time.Millisecond)
		p := bb.Min.Add(bb.Max).Div(2)
		ev := b.Scene.RenderWindow().SystemWindow.Events()
		ev.MouseButton(events.MouseDown, events.Left, p, 0)
		ev.MouseButton(events.MouseUp, events.Left, p, 0)
		assert.Eventually(t, func() bool {
			b.AsyncLock()
			defer b.AsyncUnlock()
			return !b.Scene.renderBBoxes
		}, 5*time.Second, 10*time.Millisecond, "the selection mode ends with the click")
	}, Locked(ib, func() {
		// the deepest widget clicked, the button or its text, is selected
		sels := is.treeWidget.GetSelectedNodes()
		require.Len(t, sels, 1)
		sn := sels[0].AsCoreTree().SyncNode
		assert.True(t, sn == tree.Node(target) || sn.AsTree().Parent == tree.Node(target), "%v", sn)
	}))
}

// TestInspectorWindowProfile tests opening an inspector window, which is
// recycled for the same tree, and toggling profiling.
func TestInspectorWindowProfile(t *testing.T) {
	b := NewBody("inspected")
	NewText(b).SetText("Inspect me")
	nw := len(AllRenderWindows)
	RenderImage(b, Locked(b, func() {
		InspectorWindow(b)
	}), func() {
		require.Eventually(t, func() bool { return len(AllRenderWindows) > nw+1 }, 5*time.Second, 10*time.Millisecond)
	}, Locked(b, func() {
		n := len(AllRenderWindows)
		InspectorWindow(b) // recycled
		assert.Equal(t, n, len(AllRenderWindows))
	}))

	t.Chdir(t.TempDir())
	ProfileToggle()
	assert.True(t, profile.Profiling)
	ProfileToggle()
	assert.False(t, profile.Profiling)
	for _, f := range []string{"cpu.prof", "mem.prof"} {
		_, err := os.Stat(filepath.Join(".", f))
		assert.NoError(t, err, f)
	}
}

// TestAsyncLockBeforeShown tests that the AsyncLock of a widget whose
// scene is not shown yet waits until it is shown.
func TestAsyncLockBeforeShown(t *testing.T) {
	b := NewBody()
	tx := NewText(b).SetText("later")
	locked := make(chan bool, 1)
	go func() {
		tx.AsyncLock()
		locked <- true
		tx.AsyncUnlock()
	}()
	time.Sleep(50 * time.Millisecond)
	assert.Empty(t, locked, "not shown yet")
	RenderImage(b, func() {
		select {
		case <-locked:
		case <-time.After(5 * time.Second):
			t.Error("not locked when shown")
		}
	})
}
