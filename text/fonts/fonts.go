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

// Embedded are embedded filesystems to get fonts from. By default,
// this includes a set of Noto Sans and Roboto Mono fonts. System fonts are
// automatically supported separate from this. Use [AddEmbedded] to add
// to this. This must be called before the text shaper is created to have an effect.
//
// On web, Embedded is only used for font metrics, as the actual font
// rendering happens through web fonts. See https://cogentcore.org/core/font for
// more information.
var Embedded = []fs.FS{Default}

// AddEmbedded adds to [Embedded] for font loading.
func AddEmbedded(fsys ...fs.FS) {
	Embedded = append(Embedded, fsys...)
}

// UseEmbeddedInMap adds the fonts from the current [Embedded] list to the given map.
func UseEmbeddedInMap(fontMap *fontscan.FontMap) error {
	return UseInMap(fontMap, Embedded)
}

// UseInMap adds the fonts from given file systems to the given map.
func UseInMap(fontMap *fontscan.FontMap, fss []fs.FS) error {
	var errs []error
	for _, fsys := range fss {
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
