// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fonts

import (
	"fmt"
	"io/fs"

	"cogentcore.org/core/base/errors"
	"github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/fontscan"
)

// EmbeddedFonts are embedded filesystems to get fonts from. By default,
// this includes a set of Roboto and Roboto Mono fonts. System fonts are
// automatically supported. This is not relevant on web, which uses available
// web fonts. Use [AddEmbeddedFonts] to add to this. This must be called before
// [NewShaper] to have an effect.
var EmbeddedFonts = []fs.FS{DefaultFonts}

// AddEmbeddedFonts adds to [EmbeddedFonts] for font loading.
func AddEmbeddedFonts(fsys ...fs.FS) {
	EmbeddedFonts = append(EmbeddedFonts, fsys...)
}

func UseEmbeddedFonts(fontMap *fontscan.FontMap) error {
	var errs []error
	for _, fsys := range EmbeddedFonts {
		err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				errs = append(errs, err)
				return err
			}
			if d.IsDir() {
				return nil
			}
			f, err := fsys.Open(path)
			if err != nil {
				errs = append(errs, err)
				return err
			}
			defer f.Close()
			resource, ok := f.(opentype.Resource)
			if !ok {
				err = fmt.Errorf("file %q cannot be used as an opentype.Resource", path)
				errs = append(errs, err)
				return err
			}
			err = fontMap.AddFont(resource, path, "")
			if err != nil {
				errs = append(errs, err)
				return err
			}
			return nil
		})
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
