// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package system

import (
	"image"
	"os"
	"strings"
	"testing"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/cursors"
	"cogentcore.org/core/styles/units"
	"github.com/stretchr/testify/assert"
)

func TestComputeLogicalDPI(t *testing.T) {
	// 160 * dpr * scale, rounded to a multiple of 6
	assert.Equal(t, float32(162), computeLogicalDPI(1, 1))
	assert.Equal(t, float32(318), computeLogicalDPI(2, 1))
	assert.Equal(t, float32(192), computeLogicalDPI(1, 1.2))
	assert.Equal(t, float32(480), computeLogicalDPI(2, 1.5))
}

func TestUpdateLogicalDPI(t *testing.T) {
	scale, scales := LogicalDPIScale, LogicalDPIScales
	defer func() { LogicalDPIScale, LogicalDPIScales = scale, scales }()

	LogicalDPIScale = 1
	LogicalDPIScales = nil
	sc := &Screen{Name: "main", DevicePixelRatio: 2}
	sc.UpdateLogicalDPI()
	assert.Equal(t, float32(318), sc.LogicalDPI)

	SetLogicalDPIScale("main", 1.5)
	SetLogicalDPIScale("main", 1.5) // unchanged
	assert.Equal(t, float32(1.5), LogicalDPIScales["main"])
	sc.UpdateLogicalDPI()
	assert.Equal(t, float32(480), sc.LogicalDPI, "the per-screen scale")

	other := &Screen{Name: "other", DevicePixelRatio: 1}
	other.UpdateLogicalDPI()
	assert.Equal(t, float32(162), other.LogicalDPI, "the default scale")
}

func TestScreenSizes(t *testing.T) {
	sc := &Screen{DevicePixelRatio: 2, PixelSize: image.Pt(2000, 1000), PhysicalSize: image.Pt(254, 127)}
	sc.UpdatePhysicalDPI()
	assert.Equal(t, float32(200), sc.PhysicalDPI)
	assert.Equal(t, image.Pt(200, 100), sc.WindowSizeToPixels(image.Pt(100, 50)))
	assert.Equal(t, image.Pt(100, 50), sc.WindowSizeFromPixels(image.Pt(200, 100)))
}

func TestConstrainWindowGeometry(t *testing.T) {
	scsz := image.Pt(1000, 800)
	tests := []struct {
		name              string
		pos, sz           image.Point
		wantPos, wantSize image.Point
	}{
		{"fits", image.Pt(10, 20), image.Pt(300, 200), image.Pt(10, 20), image.Pt(300, 200)},
		{"too big", image.Pt(10, 20), image.Pt(1200, 900), image.Pt(0, 0), image.Pt(1000, 800)},
		{"off the right and bottom", image.Pt(900, 700), image.Pt(300, 200), image.Pt(700, 600), image.Pt(300, 200)},
		{"negative", image.Pt(-50, -10), image.Pt(300, 200), image.Pt(0, 0), image.Pt(300, 200)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pos, sz := ConstrainWindowGeometry(tt.pos, tt.sz, scsz)
			assert.Equal(t, tt.wantPos, pos)
			assert.Equal(t, tt.wantSize, sz)
		})
	}
}

func TestGetTitle(t *testing.T) {
	var o *NewWindowOptions
	assert.Equal(t, "", o.GetTitle())
	o = &NewWindowOptions{Title: "Hello, 世界"}
	assert.Equal(t, "Hello, 世界", o.GetTitle())
	o.Title = "before\x00after"
	assert.Equal(t, "before", o.GetTitle(), "stops at NUL")
	o.Title = "ok\xffbad"
	assert.Equal(t, "ok", o.GetTitle(), "stops at invalid UTF-8")
	o.Title = strings.Repeat("a", 5000)
	assert.Len(t, o.GetTitle(), 4096)
	o.Title = strings.Repeat("a", 4095) + "世"
	assert.Equal(t, strings.Repeat("a", 4095), o.GetTitle(), "a rune cut at the limit is dropped")
}

func TestLocale(t *testing.T) {
	l := Locale("en-US")
	assert.Equal(t, "en", l.Language())
	assert.Equal(t, "US", l.Region())
	l = "fr"
	assert.Equal(t, "fr", l.Language())
	assert.Equal(t, "", l.Region())
}

func TestPlatforms(t *testing.T) {
	for _, p := range []Platforms{IOS, Android, Web} {
		assert.True(t, p.IsMobile(), p.String())
	}
	for _, p := range []Platforms{MacOS, Windows, Linux, Offscreen} {
		assert.False(t, p.IsMobile(), p.String())
	}
}

func TestGenerateHTMLArg(t *testing.T) {
	args := os.Args
	defer func() { os.Args = args }()
	os.Args = []string{"app"}
	assert.False(t, GenerateHTMLArg())
	os.Args = []string{"app", "-generatehtml"}
	assert.True(t, GenerateHTMLArg())
	os.Args = []string{"app", "-other"}
	assert.False(t, GenerateHTMLArg())
}

func TestCursorBase(t *testing.T) {
	c := &CursorBase{Vis: true}
	assert.NoError(t, c.Set(cursors.Text))
	assert.Equal(t, cursors.Text, c.Current())
	assert.True(t, c.IsVisible())
	c.Hide()
	assert.False(t, c.IsVisible())
	c.Show()
	assert.True(t, c.IsVisible())
	c.SetSize(units.Dp(48))
	assert.Equal(t, units.Dp(48), c.Size)
}

func TestClipboardBase(t *testing.T) {
	cb := &ClipboardBase{}
	assert.False(t, cb.IsEmpty())
	assert.Nil(t, cb.Read([]string{mimedata.TextPlain}))
	assert.NoError(t, cb.Write(mimedata.NewText("x")))
	cb.Clear()
}

func TestWindowFlags(t *testing.T) {
	var f WindowFlags
	f.SetFlag(true, Dialog, Modal)
	assert.True(t, f.HasFlag(Dialog))
	assert.True(t, f.HasFlag(Modal))
	assert.False(t, f.HasFlag(Fullscreen))
}
