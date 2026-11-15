// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"path/filepath"
	"testing"

	"cogentcore.org/core/base/sharedfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWindowGeometrySaveCached tests that the cached window geometries are
// saved in the file shared by all apps, merged with the geometries that
// another app saved in it since it was read.
func TestWindowGeometrySaveCached(t *testing.T) {
	skipOnWeb(t, "the window is the page, whose geometry is not saved")
	fn := filepath.Join(t.TempDir(), "window-geometry.json")
	ws := &windowGeometrySaver{inited: true}
	ws.geometries.Path = fn
	ws.resetCache()
	require.NoError(t, ws.geometries.Reload())

	// another app saves the geometry of its window
	other := sharedfile.File[screenConfigGeometries]{Path: fn}
	geom := func(x int) windowGeometries {
		return windowGeometries{Last: "s", Screens: map[string]windowGeometry{"s": {Pos: image.Pt(x, x), Size: image.Pt(100, 100)}}}
	}
	require.NoError(t, other.Update(func(g *screenConfigGeometries) {
		*g = screenConfigGeometries{"cfg": {"other": geom(1)}}
	}))

	// this app saves its window, keeping that of the other app
	ws.cache["cfg"] = map[string]windowGeometries{"this": geom(2)}
	ws.saveCached()
	assert.Empty(t, ws.cache)
	assert.Equal(t, screenConfigGeometries{"cfg": {"other": geom(1), "this": geom(2)}}, ws.geometries.Data)
	require.NoError(t, other.Reload())
	assert.Equal(t, ws.geometries.Data, other.Data)
}

// TestWindowGeometriesForScreen tests getting the saved geometry of a
// window for an active screen, and the helpers of window geometries.
func TestWindowGeometriesForScreen(t *testing.T) {
	skipOnWeb(t, "the window is the page, whose geometry is not saved")
	sc := TheApp.Screen(0)
	wg := windowGeometry{Size: image.Pt(100, 80), Pos: image.Pt(5, 5), DPI: 96}
	wgs := windowGeometries{Last: sc.Name, Screens: map[string]windowGeometry{sc.Name: wg}}
	got, gsc := wgs.getForScreen("")
	require.NotNil(t, got)
	assert.Equal(t, wg, *got, "the last screen")
	assert.Same(t, sc, gsc)

	wgs.Last = "gone"
	got, _ = wgs.getForScreen(sc.Name)
	require.NotNil(t, got, "the given screen")
	assert.Equal(t, wg, *got)
	got, _ = wgs.getForScreen("other")
	require.NotNil(t, got, "any active screen")

	wgs.Last = sc.Name
	wgs.Screens = map[string]windowGeometry{"other": wg}
	got, _ = wgs.getForScreen("")
	assert.Nil(t, got, "no geometry for the last screen or any active screen")
	wgs.Screens = map[string]windowGeometry{}
	got, _ = wgs.getForScreen(sc.Name)
	assert.Nil(t, got)

	assert.Contains(t, wg.String(), "Size: (100,80)")
	big := windowGeometry{Size: image.Pt(100000, 100000), Pos: image.Pt(-50, -50)}
	big.constrainGeom(sc)
	assert.LessOrEqual(t, big.Size.X, sc.PixelSize.X, "constrained to the screen")
	assert.GreaterOrEqual(t, big.Pos.X, 0)

	ws := &windowGeometrySaver{inited: true}
	ws.geometries.Path = filepath.Join(t.TempDir(), "window-geometry.json")
	ws.resetCache()
	ws.geometries.Data = screenConfigGeometries{"cfg": {"w": wgs}}
	ws.deleteAll()
	assert.Nil(t, ws.geometries.Data)
}
