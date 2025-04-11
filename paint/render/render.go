// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package render

import (
	"reflect"
	"slices"

	"cogentcore.org/core/base/reflectx"
)

// Render is the sequence of painting [Item]s recorded
// from a [paint.Painter]
type Render []Item

// Clone returns a copy of this Render,
// with shallow clones of the Items and Renderers lists.
func (pr *Render) Clone() Render {
	return slices.Clone(*pr)
}

// Add adds item(s) to render. Filters any nil items.
func (pr *Render) Add(item ...Item) *Render {
	for _, it := range item {
		if reflectx.IsNil(reflect.ValueOf(it)) {
			continue
		}
		*pr = append(*pr, it)
	}
	return pr
}

// Reset resets back to an empty Render state.
// It preserves the existing slice memory for re-use.
func (pr *Render) Reset() {
	*pr = (*pr)[:0]
}
