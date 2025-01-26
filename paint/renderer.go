// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package paint

// Render represents a collection of render [Item]s to be rendered.
type Render []Item

// Item is a union interface for render items: path.Path, text.Text, or Image.
type Item interface {
	isRenderItem()
}
