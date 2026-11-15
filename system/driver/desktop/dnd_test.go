// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package desktop

import (
	"bytes"
	"image"
	"image/png"
	"testing"
	"time"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"github.com/go-gl/glfw/v3.4/glfw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDragData tests the data and image provided to other apps for a drag.
func TestDragData(t *testing.T) {
	md := mimedata.NewTextPlus("hello", mimedata.DataJson, []byte(`{"a":1}`))
	mime, text := dragData(md, nil)
	assert.Equal(t, "hello", string(text))
	isMulti, _, _, _ := mimedata.IsMultipart(mime)
	assert.True(t, isMulti)
	mime, text = dragData(md, []string{"/tmp/f.txt"})
	assert.Nil(t, mime, "files are dragged as files")
	assert.Nil(t, text)

	assert.Nil(t, dragImagePNG(nil))
	b := dragImagePNG(image.NewRGBA(image.Rect(0, 0, 4, 3)))
	img, err := png.Decode(bytes.NewReader(b))
	require.NoError(t, err)
	assert.Equal(t, image.Rect(0, 0, 4, 3), img.Bounds())
}

// TestDragWindow tests drags over and drops on a window from other apps,
// through the registry of the native handles of the windows.
func TestDragWindow(t *testing.T) {
	w := newTestWindow(t)
	addDragWindow(12345, w)
	assert.Same(t, w, dragWindow(12345))
	deleteDragWindow(12345)
	assert.Nil(t, dragWindow(12345))
	drainEvents(w, 50*time.Millisecond)

	w.dragMove(false, image.Pt(5, 6), 0, false)
	w.dragMove(true, image.Pt(5, 6), 0, true)
	w.dragDrop(image.Pt(7, 8), 0, false, nil, nil) // nothing to drop
	w.dragDrop(image.Pt(7, 8), 0, false, []string{"/tmp/a", "/tmp/b"}, nil)
	md := mimedata.NewTextPlus("hello", mimedata.DataJson, []byte(`{"a":1}`))
	w.dragDrop(image.Pt(9, 9), 0, true, nil, md.ToMultipart())

	var types []events.Types
	var drops []*events.DragDrop
	for _, e := range drainEvents(w, 100*time.Millisecond) {
		types = append(types, e.Type())
		if d, ok := e.(*events.DragDrop); ok && e.Type() == events.Drop {
			drops = append(drops, d)
		}
	}
	assert.Contains(t, types, events.DragMove)
	assert.Contains(t, types, events.DragLeave)
	require.Len(t, drops, 2)
	assert.Len(t, drops[0].Data, 2, "the files")
	assert.Len(t, drops[1].Data, 2, "the mime data of another Cogent Core app")
}

// TestGlfwKeyCodes tests that all of the glfw keys of the range that it
// defines have distinct key codes, other than the ones that have none.
func TestGlfwKeyCodes(t *testing.T) {
	seen := map[key.Codes]glfw.Key{}
	n := 0
	for k := glfw.KeySpace; k <= glfw.KeyLast; k++ {
		c := GlfwKeyCode(k)
		if c == key.CodeUnknown {
			continue
		}
		n++
		if prev, ok := seen[c]; ok {
			t.Errorf("glfw keys %d and %d both have the code %v", prev, k, c)
		}
		seen[c] = k
	}
	assert.Greater(t, n, 100)
}
