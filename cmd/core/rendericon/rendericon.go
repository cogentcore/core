// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rendericon

import (
	"errors"
	"fmt"
	"image"
	"io/fs"
	"strings"

	"cogentcore.org/core/icons"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/svg"
)

// Render renders the icon located at icon.svg at the given size.
// If no such icon exists, it sets it to a placeholder icon, a blue version of
// [icons.Toolbar].
func Render(size int) (*image.RGBA, error) {
	paint.FontLibrary.InitFontPaths(paint.FontPaths...)

	sv := svg.NewSVG(size, size)

	spath := "icon.svg"
	err := sv.OpenXML(spath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("error opening svg icon file: %w", err)
		}
		err = sv.ReadXML(strings.NewReader(icons.DefaultAppIcon))
		if err != nil {
			return nil, err
		}
	}

	sv.Render()
	return sv.Pixels, nil
}
