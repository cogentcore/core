// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"image"
	"testing"

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
