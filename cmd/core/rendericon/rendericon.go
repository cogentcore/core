// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rendericon

import (
	"errors"
	"fmt"
	"image"
	"io/fs"
	"os"
	"strings"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/math32"
	_ "cogentcore.org/core/paint/renderers"
	"cogentcore.org/core/svg"
)

// Render renders the icon located at icon.svg at the given size.
// If no such icon exists, it sets it to a placeholder icon, [icons.DefaultAppIcon].
func Render(size int) (*image.RGBA, error) {
	sv := svg.NewSVG(math32.Vec2(float32(size), float32(size)))

	spath := "icon.svg"
	err := sv.OpenXML(spath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("error opening svg icon file: %w", err)
		}
		err = os.WriteFile(spath, []byte(icons.CogentCore), 0666)
		if err != nil {
			return nil, err
		}
		err = sv.ReadXML(strings.NewReader(string(icons.CogentCore)))
		if err != nil {
			return nil, err
		}
	}

	return imagex.AsRGBA(sv.RenderImage()), nil
}
