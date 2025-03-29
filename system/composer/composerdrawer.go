// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package composer

// ComposerDrawer implements [Composer] using a [Drawer].
type ComposerDrawer struct {

	// Drawer is the [Drawer] used for composition.
	Drawer Drawer

	// Sources are the composition [Source]s.
	Sources []Source
}

func (cd *ComposerDrawer) Start() {
	cd.Sources = cd.Sources[:0]
}

func (cd *ComposerDrawer) Add(s Source, ctx any) {
	if s == nil { // TODO: necessary?
		return
	}
	cd.Sources = append(cd.Sources, s)
}

func (cd *ComposerDrawer) Compose() {
	cd.Drawer.Start()
	for _, s := range cd.Sources {
		s.Draw(cd)
	}
	cd.Drawer.End()
}

func (cd *ComposerDrawer) Redraw() {
	cd.Drawer.Redraw()
}
