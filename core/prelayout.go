// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
)

// PreLayout returns a function that styles and lays out a widget that is
// in no scene (as the new content of a page, made in another goroutine),
// as it would be as the only child of the given parent widget in this
// scene, in a scene of its own that shares the text shaper of the window
// of this scene, and so its caches. Laying the widget out again once it is
// added to the parent (with the lock of this scene held) is then much
// quicker, as the texts in it are shaped and wrapped from the caches.
//
// PreLayout is called with the lock of the scene held (as it gets the
// state of the scene and the parent), and the function that it returns is
// called without it, in any goroutine, before the widget is added to the
// parent. It returns nil if the scene is not in a window yet.
func (sc *Scene) PreLayout(parent Widget) func(w Widget) {
	rc := sc.renderContext()
	if rc == nil || rc.textShaper == nil {
		return nil
	}
	off := &renderContext{logicalDPI: rc.logicalDPI, textShaper: rc.textShaper, visible: true}
	sceneSize := sc.SceneGeom.Size
	pwb := parent.AsWidget()
	pst := pwb.Styles // a copy, for the inherited styles
	size := pwb.Geom.Size.Alloc.Content
	return func(w Widget) {
		psc := NewScene("pre-layout")
		psc.offscreen = off
		psc.SceneGeom.Size = sceneSize
		psc.Styler(func(s *styles.Style) {
			s.Padding.Zero()
			s.Gap.Zero()
		})
		host := NewFrame(psc) // as the parent
		host.Styler(func(s *styles.Style) {
			// the styles of the parent, which its children inherit, in the
			// size of its content
			st := s.State
			*s = pst
			s.State = st
			s.Padding.Zero()
			s.Margin.Zero()
			s.Border.Width.Zero()
			s.Min.Set(units.Dot(size.X), units.Dot(size.Y))
			s.Max.Set(units.Dot(size.X), units.Dot(size.Y))
			s.Overflow.Set(styles.OverflowAuto)
		})
		host.AddChild(w)
		psc.StyleTree()
		psc.LayoutScene()
		// the widget is taken out of the scene, to be added to the parent
		host.Children = nil
		w.AsTree().Parent = nil
	}
}
