// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shapedgt

import (
	"fmt"
	"os"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/text/text"
	"github.com/go-text/typesetting/fontscan"
)

// FontList returns the list of fonts that have been loaded.
func (sh *Shaper) FontList() []shaped.FontInfo {
	str := errors.Log1(os.UserCacheDir())
	ft := errors.Log1(fontscan.SystemFonts(nil, str))
	fi := make([]shaped.FontInfo, len(ft))
	for i := range ft {
		fi[i].Family = ft[i].Family
		as := ft[i].Aspect
		fi[i].Weight = rich.Weights(int(as.Weight / 100.0))
		fi[i].Slant = rich.Slants(as.Style - 1)
		// fi[i].Stretch = rich.Stretch() // not avail
		fi[i].Stretch = rich.StretchNormal
	}
	return fi
}

func (sh *Shaper) FontDebug() {
	tsty := text.NewStyle()
	fmt.Println("Font Families:")
	for fam := rich.SansSerif; fam < rich.Custom; fam++ {
		sty := rich.NewStyle().SetFamily(fam)
		tx := rich.NewText(sty, []rune("Test font"))
		out := sh.Shape(tx, tsty, &rich.DefaultSettings)
		oface := out[0].(*Run).Output.Face.Describe()
		fmt.Println(fam, "settings:", tsty.FontFamily(sty), "### Actual:", oface.Family, oface)
	}
}
