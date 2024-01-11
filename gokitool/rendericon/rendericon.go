// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rendericon

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io/fs"
	"os"
	"path/filepath"

	"goki.dev/colors"
	"goki.dev/icons"
	"goki.dev/paint"
	"goki.dev/svg"
)

// Render renders the icon located at .goki/icon.svg at the given size.
// If no such icon exists, it sets it to a placeholder icon, a blue version of
// [icons.Toolbar].
func Render(size int) (*image.RGBA, error) {
	paint.FontLibrary.InitFontPaths(paint.FontPaths...)

	sv := svg.NewSVG(size, size)
	sv.Norm = true
	sv.Color = colors.C(colors.FromRGB(66, 133, 244)) // Google Blue (#4285f4)

	spath := filepath.Join(".goki", "icon.svg")
	err := sv.OpenXML(spath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("error opening svg icon file: %w", err)
		}
		ic, err := fs.ReadFile(icons.Icons, icons.Toolbar.Filename())
		if err != nil {
			return nil, err
		}
		err = os.WriteFile(spath, ic, 0666)
		if err != nil {
			return nil, err
		}
		err = sv.ReadXML(bytes.NewReader(ic))
		if err != nil {
			return nil, err
		}
	}

	sv.Render()
	return sv.Pixels, nil
}
