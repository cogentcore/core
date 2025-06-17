// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cursors

import (
	"bytes"
	"fmt"
	"image"
	"io/fs"

	"cogentcore.org/core/colors"
)

// SVG reads the cursor with the given name and returns a byte slice of SVG data.
// This is mainly used by [cogentcore.org/core/cursors/cursorimg], but is also used
// by the docs in an example. This function performs color replacement as described
// in [cogentcore.org/core/cursors/cursorimg.Get].
func SVG(name string) ([]byte, error) {
	b, err := fs.ReadFile(Cursors, "svg/"+name+".svg")
	if err != nil {
		return nil, err
	}
	b = replaceColors(b)
	return b, nil
}

// replaceColors replaces literal cursor colors in the given SVG with scheme colors.
func replaceColors(b []byte) []byte {
	m := map[string]image.Image{
		"#fff": colors.Palette.Neutral.ToneUniform(100),
		"#000": colors.Palette.Neutral.ToneUniform(0),
		"#f00": colors.Scheme.Error.Base,
		"#0f0": colors.Scheme.Success.Base,
		"#ff0": colors.Scheme.Warn.Base,
	}
	for old, clr := range m {
		b = bytes.ReplaceAll(b, []byte(fmt.Sprintf("%q", old)), []byte(fmt.Sprintf("%q", colors.AsHex(colors.ToUniform(clr)))))
	}
	return b
}
