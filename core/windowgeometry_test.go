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
