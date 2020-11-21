// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import "github.com/goki/gi/gist"

// This file contains all the Name types that drive chooser menus when they
// show up as fields or args, using the giv ValueView system.

// Color is the GoGi version of RGBA color with special methods.
// This is an alias for the version defined in the gist styles.
type Color = gist.Color

// ColorName provides a value-view GUI lookup of valid color names
type ColorName string

// FontName is used to specify a font, as the unique name of the font family.
// This automatically provides a chooser menu for fonts using giv ValueView.
type FontName string
