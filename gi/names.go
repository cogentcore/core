// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

// This file contains all the Name types that drive chooser menus when they
// show up as fields or args, using the giv Value system.

// ColorName provides a value-view GUI lookup of valid color names
type ColorName string

func (cn ColorName) String() string {
	return string(cn)
}

// FontName is used to specify a font, as the unique name of
// the font family.
// This automatically provides a chooser menu for fonts
// using giv Value.
type FontName string

func (fn FontName) String() string {
	return string(fn)
}

// FileName is used to specify an filename (including path).
// Automatically opens the FileView dialog using Value system.
// Use this for any method args that are filenames to trigger
// use of FileViewDialog under FuncButton automatic method calling.
type FileName string

func (fn FileName) String() string {
	return string(fn)
}

// HiStyleName is a highlighting style name
type HiStyleName string

func (hs HiStyleName) String() string {
	return string(hs)
}
