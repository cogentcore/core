// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shapedgt

import (
	"os"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	"github.com/go-text/typesetting/fontscan"
)

// FontList returns the list of fonts that have been loaded.
func (sh *Shaper) FontList() []shaped.FontInfo {
	str := errors.Log1(os.UserCacheDir())
	ft := errors.Log1(fontscan.SystemFonts(nil, str))
	fi := make([]shaped.FontInfo, len(ft))
	for i := range ft {
		fi[i].Name = ft[i].Family
		as := ft[i].Aspect
		fi[i].Weight = rich.Weights(int(as.Weight / 100.0))
		fi[i].Slant = rich.Slants(as.Style - 1)
		// fi[i].Stretch = rich.Stretch() // not avail
		fi[i].Stretch = rich.StretchNormal
		fi[i].Example = shaped.FontInfoExample
	}
	return fi
}
