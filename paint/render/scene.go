// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package render

// Scene is an ordered list of [Render] elements, that
// can all be rendered and drawn to a [system.Drawer],
// in a separate goroutine for efficiency.
type Scene []Render

func (sc *Scene) Add(r Render) Scene {
	(*sc) = append(*sc, r)
	return *sc
}

func (sc *Scene) Reset() Scene {
	(*sc) = (*sc)[:0]
	return *sc
}
