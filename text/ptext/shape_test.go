// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ptext

import (
	"fmt"
	"testing"

	"cogentcore.org/core/base/runes"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
)

func TestSpans(t *testing.T) {
	src := "The lazy fox typed in some familiar text"
	sr := []rune(src)
	sp := rich.Spans{}
	plain := rich.NewStyle()
	ital := rich.NewStyle().SetSlant(rich.Italic)
	ital.SetStrokeColor(colors.Red)
	boldBig := rich.NewStyle().SetWeight(rich.Bold).SetSize(1.5)
	sp.Add(plain, sr[:4])
	sp.Add(ital, sr[4:8])
	fam := []rune("familiar")
	ix := runes.Index(sr, fam)
	sp.Add(plain, sr[8:ix])
	sp.Add(boldBig, sr[ix:ix+8])
	sp.Add(plain, sr[ix+8:])

	ctx := &rich.Context{}
	ctx.Defaults()
	uc := units.Context{}
	uc.Defaults()
	ctx.ToDots(&uc)
	sh := NewShaper()
	runs := sh.Shape(sp, ctx)
	fmt.Println(runs)
}
