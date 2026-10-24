// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"
	"testing"
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
)

// imagelessRenderer is a renderer without an image, as the HTML canvas
// renderer of the web.
type imagelessRenderer struct{ size math32.Vector2 }

func (r *imagelessRenderer) Render(render.Render) render.Renderer { return r }
func (r *imagelessRenderer) Size() (units.Units, math32.Vector2) {
	return units.UnitDot, r.size
}
func (r *imagelessRenderer) SetSize(un units.Units, size math32.Vector2) { r.size = size }
func (r *imagelessRenderer) Image() image.Image                          { return nil }
func (r *imagelessRenderer) Source() []byte                              { return nil }

// A scene is restyled by its next update after a resize, once however many
// times it is resized before it, and a resize to its size does nothing, also
// with a renderer that has no image.
func TestSceneResizeRestyle(t *testing.T) {
	b := NewBody()
	styled := 0
	b.Styler(func(s *styles.Style) { styled++ })
	sc := b.Scene
	sc.Renderer = &imagelessRenderer{}
	rc := newRenderContext()
	sc.lastRender.saveRender(rc)

	geom := math32.Geom2DInt{Size: image.Pt(300, 200)}
	assert.True(t, sc.Resize(geom))
	assert.False(t, sc.Resize(geom))
	geom.Size.X = 400
	assert.True(t, sc.Resize(geom))
	assert.Equal(t, 0, styled)
	assert.True(t, sc.lastRender.needsRestyle(rc))
	assert.True(t, sc.hasFlag(sceneNeedsLayout))

	// The next update styles the scene once.
	sc.lastRender.saveRender(rc)
	assert.False(t, sc.lastRender.needsRestyle(rc))
}

// Resizing a scene to its content sizes it in another goroutine, and the
// timeout of a snackbar closes it in a timer, while the event loop and
// other goroutines update the widgets of the scene, all with the render
// lock, so that the race detector finds no races on the plain flags of
// the scene and its widgets.
func TestResizeToContentAsync(t *testing.T) {
	b := NewBody()
	bt := NewButton(b).SetText("Resize")
	tx := NewText(b).SetText("Text")
	bt.OnClick(func(e events.Event) {
		// As in an event handler, with the render lock held.
		b.Scene.ResizeToContent()
		tx.SetText("Clicked").Update()
	})
	b.runAndShowNewWindow()
	rw := b.Scene.RenderWindow()
	sc := b.Scene
	for i := range 20 {
		b.AsyncLock()
		bt.Send(events.Click)
		NewBody().AddSnackbarText("Snackbar").NewSnackbar(bt).SetTimeout(time.Millisecond).Run()
		b.AsyncUnlock()
		b.AsyncLock()
		sc.ResizeToContent()
		tx.SetText(fmt.Sprintf("Text %d", i)).Update()
		bt.NeedsRender()
		b.AsyncUnlock()
		time.Sleep(time.Millisecond)
	}
	// The resizes of the goroutines above are done once we have their lock,
	// and the snackbar timers are done after waiting for them.
	time.Sleep(50 * time.Millisecond)
	rw.resizeMu.Lock()
	rw.resizeMu.Unlock()
	b.waitNoEvents(rw)
	rw.waitRenderDone()
	rc := sc.renderContext()
	rc.Lock()
	assert.False(t, sc.hasFlag(sceneContentSizing))
	b.Close()
	rc.Unlock()
}

// A function that [WidgetBase.runInWindow] sends to the event loop of a
// window runs there, with the render lock.
func TestRunInWindowEvent(t *testing.T) {
	b := NewBody()
	tx := NewText(b).SetText("Text")
	b.runAndShowNewWindow()
	rw := b.Scene.RenderWindow()
	done := make(chan struct{})
	rw.SystemWindow.Events().Custom(windowFunc(func() {
		tx.SetText("Updated").Update()
		close(done)
	}))
	<-done
	rc := b.Scene.renderContext()
	rc.Lock()
	assert.Equal(t, "Updated", tx.Text)
	b.Close()
	rc.Unlock()
}

// A function that [runInCurrentWindow] sends from a goroutine without a lock,
// as the snackbar for an update of a web app from the browser, runs in the
// event loop of the current window, with its lock.
func TestRunInCurrentWindow(t *testing.T) {
	b := NewBody()
	b.runAndShowNewWindow()
	rw := b.Scene.RenderWindow()
	done := make(chan *Stage)
	go runInCurrentWindow(func() {
		done <- NewBody().AddSnackbarText("A new version is available").
			NewSnackbar(nil).SetTimeout(0).Run()
	})
	st := <-done
	rc := b.Scene.renderContext()
	rc.Lock()
	assert.Same(t, &rw.mains, st.Mains)
	st.ClosePopup()
	b.Close()
	rc.Unlock()
}
