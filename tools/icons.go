// Copyright (c) 2023, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tools

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"goki.dev/colors"
	"goki.dev/goki/config"
	"goki.dev/icons"
	"goki.dev/svg"
)

var IconSizes = []int{16, 32}

// Icons converts the icon located at .goki/icons/svg.svg into various
// different formats and sizes necessary for app packaging. If no such
// icon exists, it sets it to a placeholder icon, a blue version of
// [icons.SelectWindow]. It is run automatically for apps (not libraries)
// in goki init.
func Icons(c *config.Config) error { //gti:add
	sv := svg.NewSVG(0, 0)
	spath := filepath.Join(".goki", "icons", "svg.svg")
	err := sv.OpenXML(spath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("error opening svg icon file: %w", err)
		}
		ic, err := fs.ReadFile(icons.Icons, icons.SelectWindow.Filename())
		if err != nil {
			return err
		}
		err = os.WriteFile(spath, ic, 0666)
		if err != nil {
			return err
		}
		err = sv.ReadXML(bytes.NewReader(ic))
		if err != nil {
			return err
		}
		sv.Color = colors.C(colors.FromRGB(66, 133, 244)) // Google Blue (#4285f4)
	}
	for _, sz := range IconSizes {
		sv.Resize(image.Pt(sz, sz))
		err := sv.SavePNG(filepath.Join(".goki", "icons", strconv.Itoa(sz)+".png"))
		if err != nil {
			return err
		}
	}
	return nil
}
