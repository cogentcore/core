// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package shapedgt

import (
	"embed"
	"fmt"
	"io/fs"
	"os"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/fontscan"
	"github.com/go-text/typesetting/shaping"
	"golang.org/x/image/math/fixed"
)

// Shaper is the text shaper and wrapper, from go-text/shaping.
type Shaper struct {
	shaper   shaping.HarfbuzzShaper
	wrapper  shaping.LineWrapper
	fontMap  *fontscan.FontMap
	splitter shaping.Segmenter

	// outBuff is the output buffer to avoid excessive memory consumption.
	outBuff []shaping.Output
}

// EmbeddedFonts are embedded filesystems to get fonts from. By default,
// this includes a set of Roboto and Roboto Mono fonts. System fonts are
// automatically supported. This is not relevant on web, which uses available
// web fonts. Use [AddEmbeddedFonts] to add to this. This must be called before
// [NewShaper] to have an effect.
var EmbeddedFonts = []fs.FS{defaultFonts}

// AddEmbeddedFonts adds to [EmbeddedFonts] for font loading.
func AddEmbeddedFonts(fsys ...fs.FS) {
	EmbeddedFonts = append(EmbeddedFonts, fsys...)
}

//go:embed fonts/*.ttf
var defaultFonts embed.FS

// todo: per gio: systemFonts bool, collection []FontFace
func NewShaper() *Shaper {
	sh := &Shaper{}
	sh.fontMap = fontscan.NewFontMap(nil)
	// TODO(text): figure out cache dir situation (especially on mobile and web)
	str, err := os.UserCacheDir()
	if errors.Log(err) != nil {
		// slog.Printf("failed resolving font cache dir: %v", err)
		// shaper.logger.Printf("skipping system font load")
	}
	// fmt.Println("cache dir:", str)
	if err := sh.fontMap.UseSystemFonts(str); err != nil {
		errors.Log(err)
		// shaper.logger.Printf("failed loading system fonts: %v", err)
	}
	for _, fsys := range EmbeddedFonts {
		errors.Log(fs.WalkDir(fsys, "fonts", func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			f, err := fsys.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			resource, ok := f.(opentype.Resource)
			if !ok {
				return fmt.Errorf("file %q cannot be used as an opentype.Resource", path)
			}
			err = sh.fontMap.AddFont(resource, path, "")
			if err != nil {
				return err
			}
			return nil
		}))
	}
	// for _, f := range collection {
	// 	shaper.Load(f)
	// 	shaper.defaultFaces = append(shaper.defaultFaces, string(f.Font.Typeface))
	// }
	sh.shaper.SetFontCacheSize(32)
	return sh
}

// Shape turns given input spans into [Runs] of rendered text,
// using given context needed for complete styling.
// The results are only valid until the next call to Shape or WrapParagraph:
// use slices.Clone if needed longer than that.
func (sh *Shaper) Shape(tx rich.Text, tsty *text.Style, rts *rich.Settings) []shaping.Output {
	return sh.shapeText(tx, tsty, rts, tx.Join())
}

// shapeText implements Shape using the full text generated from the source spans
func (sh *Shaper) shapeText(tx rich.Text, tsty *text.Style, rts *rich.Settings, txt []rune) []shaping.Output {
	if tx.Len() == 0 {
		return nil
	}
	sty := rich.NewStyle()
	sh.outBuff = sh.outBuff[:0]
	for si, s := range tx {
		in := shaping.Input{}
		start, end := tx.Range(si)
		rs := sty.FromRunes(s)
		if len(rs) == 0 {
			continue
		}
		q := StyleToQuery(sty, rts)
		sh.fontMap.SetQuery(q)

		in.Text = txt
		in.RunStart = start
		in.RunEnd = end
		in.Direction = goTextDirection(sty.Direction, tsty)
		fsz := tsty.FontSize.Dots * sty.Size
		in.Size = math32.ToFixed(fsz)
		in.Script = rts.Script
		in.Language = rts.Language

		ins := sh.splitter.Split(in, sh.fontMap) // this is essential
		for _, in := range ins {
			if in.Face == nil {
				fmt.Println("nil face in input", len(rs), string(rs))
				// fmt.Printf("nil face for in: %#v\n", in)
				continue
			}
			o := sh.shaper.Shape(in)
			sh.outBuff = append(sh.outBuff, o)
		}
	}
	return sh.outBuff
}

// goTextDirection gets the proper go-text direction value from styles.
func goTextDirection(rdir rich.Directions, tsty *text.Style) di.Direction {
	dir := tsty.Direction
	if rdir != rich.Default {
		dir = rdir
	}
	return dir.ToGoText()
}

// todo: do the paragraph splitting!  write fun in rich.Text

// DirectionAdvance advances given position based on given direction.
func DirectionAdvance(dir di.Direction, pos fixed.Point26_6, adv fixed.Int26_6) fixed.Point26_6 {
	if dir.IsVertical() {
		pos.Y += -adv
	} else {
		pos.X += adv
	}
	return pos
}

// StyleToQuery translates the rich.Style to go-text fontscan.Query parameters.
func StyleToQuery(sty *rich.Style, rts *rich.Settings) fontscan.Query {
	q := fontscan.Query{}
	q.Families = rich.FamiliesToList(sty.FontFamily(rts))
	q.Aspect = StyleToAspect(sty)
	return q
}

// StyleToAspect translates the rich.Style to go-text font.Aspect parameters.
func StyleToAspect(sty *rich.Style) font.Aspect {
	as := font.Aspect{}
	as.Style = font.Style(1 + sty.Slant)
	as.Weight = font.Weight(sty.Weight.ToFloat32())
	as.Stretch = font.Stretch(sty.Stretch.ToFloat32())
	return as
}
