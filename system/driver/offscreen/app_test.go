// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package offscreen

import (
	"image"
	"image/color"
	"image/draw"
	"os"
	"testing"

	"cogentcore.org/core/events"
	"cogentcore.org/core/system"
	"cogentcore.org/core/system/composer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	Init()
	os.Exit(m.Run())
}

func TestAppScreen(t *testing.T) {
	assert.Equal(t, system.Offscreen, TheApp.Platform())
	assert.DirExists(t, TheApp.DataDir())
	assert.Same(t, TheApp, system.TheApp.(*App))
	sc := TheApp.Screen(0)
	assert.Equal(t, image.Pt(800, 600), sc.PixelSize)
	assert.Equal(t, float32(1), sc.DevicePixelRatio)
	assert.Equal(t, float32(160), sc.LogicalDPI)
	assert.Equal(t, image.Pt(127, 95), sc.PhysicalSize, "in mm at 160 dpi")

	sc.PixelSize = image.Pt(1000, 500) // a custom size is kept
	TheApp.GetScreens()
	assert.Equal(t, image.Pt(1000, 500), sc.Geometry.Max)
	sc.PixelSize = image.Point{}
	TheApp.GetScreens()
	assert.Equal(t, image.Pt(800, 600), sc.PixelSize)
}

func TestNewWindowEvents(t *testing.T) {
	sw, err := TheApp.NewWindow(&system.NewWindowOptions{Size: image.Pt(320, 240)})
	require.NoError(t, err)
	w := sw.(*Window)
	defer w.Close()
	assert.Same(t, TheApp.Screen(0), w.Screen())
	assert.Equal(t, image.Pt(320, 240), w.Size())

	// the window is shown with these events, in this order
	var types []events.Types
	for len(types) < 4 {
		e := w.Events().Deque.NextEvent()
		if e.Type() != events.WindowPaint {
			types = append(types, e.Type())
		}
	}
	assert.ElementsMatch(t, []events.Types{events.WindowResize, events.Window, events.Window, events.Window}, types)
}

// fillSource fills its rectangle with a color.
type fillSource struct {
	r   image.Rectangle
	clr color.RGBA
}

func (fs *fillSource) Draw(c composer.Composer) {
	cd := c.(*composer.ComposerDrawer)
	cd.Drawer.Copy(fs.r.Min, image.NewUniform(fs.clr), fs.r, draw.Src, composer.Unchanged)
}

func TestDrawerCompose(t *testing.T) {
	sw, err := TheApp.NewWindow(&system.NewWindowOptions{Size: image.Pt(40, 30)})
	require.NoError(t, err)
	w := sw.(*Window)
	defer w.Close()
	red := color.RGBA{255, 0, 0, 255}
	c := w.Composer()
	c.Start()
	c.Add(&fillSource{image.Rect(10, 10, 20, 20), red}, nil)
	c.Compose()
	c.Redraw()
	img := w.Compose.Drawer.(*Drawer).GetImage()
	require.NotNil(t, img)
	assert.Equal(t, image.Rect(0, 0, 40, 30), img.Rect, "the size of the window")
	assert.Equal(t, red, img.RGBAAt(15, 15))
	assert.Equal(t, color.RGBA{}, img.RGBAAt(25, 25))
	assert.Nil(t, w.Compose.Drawer.Renderer())

	w.SetSize(image.Pt(50, 50))
	assert.Equal(t, image.Pt(50, 50), w.WinSize())
	w.GeomMu.Lock()
	w.PixelSize = image.Pt(50, 50) // as by a resize of the system window
	w.GeomMu.Unlock()
	c.Start()
	c.Compose()
	assert.Equal(t, image.Rect(0, 0, 50, 50), w.Compose.Drawer.(*Drawer).GetImage().Rect, "a new image for a new size")
}

func TestWindowClose(t *testing.T) {
	sw, err := TheApp.NewWindow(&system.NewWindowOptions{Size: image.Pt(40, 30)})
	require.NoError(t, err)
	n := TheApp.NWindows()
	sw.Close()
	assert.Equal(t, n-1, TheApp.NWindows())
}
