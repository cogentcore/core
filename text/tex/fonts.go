// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// note: adapted from https://github.com/tdewolff/canvas,
// Copyright (c) 2015 Taco de Wolff, under an MIT License.
// and gioui: Unlicense OR MIT, Copyright (c) 2019 The Gio authors

package tex

import (
	"bytes"
	"fmt"
	"strconv"
	"sync"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"github.com/go-fonts/latin-modern/lmmath"
	"github.com/go-fonts/latin-modern/lmmono10italic"
	"github.com/go-fonts/latin-modern/lmmono10regular"
	"github.com/go-fonts/latin-modern/lmmono12regular"
	"github.com/go-fonts/latin-modern/lmmono8regular"
	"github.com/go-fonts/latin-modern/lmmono9regular"
	"github.com/go-fonts/latin-modern/lmmonocaps10regular"
	"github.com/go-fonts/latin-modern/lmmonoslant10regular"
	"github.com/go-fonts/latin-modern/lmroman10bold"
	"github.com/go-fonts/latin-modern/lmroman10bolditalic"
	"github.com/go-fonts/latin-modern/lmroman10italic"
	"github.com/go-fonts/latin-modern/lmroman10regular"
	"github.com/go-fonts/latin-modern/lmroman12bold"
	"github.com/go-fonts/latin-modern/lmroman12italic"
	"github.com/go-fonts/latin-modern/lmroman12regular"
	"github.com/go-fonts/latin-modern/lmroman17regular"
	"github.com/go-fonts/latin-modern/lmroman5bold"
	"github.com/go-fonts/latin-modern/lmroman5regular"
	"github.com/go-fonts/latin-modern/lmroman6bold"
	"github.com/go-fonts/latin-modern/lmroman6regular"
	"github.com/go-fonts/latin-modern/lmroman7bold"
	"github.com/go-fonts/latin-modern/lmroman7italic"
	"github.com/go-fonts/latin-modern/lmroman7regular"
	"github.com/go-fonts/latin-modern/lmroman8bold"
	"github.com/go-fonts/latin-modern/lmroman8italic"
	"github.com/go-fonts/latin-modern/lmroman8regular"
	"github.com/go-fonts/latin-modern/lmroman9bold"
	"github.com/go-fonts/latin-modern/lmroman9italic"
	"github.com/go-fonts/latin-modern/lmroman9regular"
	"github.com/go-fonts/latin-modern/lmromancaps10regular"
	"github.com/go-fonts/latin-modern/lmromandunh10regular"
	"github.com/go-fonts/latin-modern/lmromanslant10bold"
	"github.com/go-fonts/latin-modern/lmromanslant10regular"
	"github.com/go-fonts/latin-modern/lmromanslant12regular"
	"github.com/go-fonts/latin-modern/lmromanslant17regular"
	"github.com/go-fonts/latin-modern/lmromanslant8regular"
	"github.com/go-fonts/latin-modern/lmromanslant9regular"
	"github.com/go-fonts/latin-modern/lmromanunsl10regular"
	"github.com/go-fonts/latin-modern/lmsans10bold"
	"github.com/go-fonts/latin-modern/lmsans10oblique"
	"github.com/go-fonts/latin-modern/lmsans10regular"
	"github.com/go-fonts/latin-modern/lmsans12oblique"
	"github.com/go-fonts/latin-modern/lmsans12regular"
	"github.com/go-fonts/latin-modern/lmsans17oblique"
	"github.com/go-fonts/latin-modern/lmsans17regular"
	"github.com/go-fonts/latin-modern/lmsans8oblique"
	"github.com/go-fonts/latin-modern/lmsans8regular"
	"github.com/go-fonts/latin-modern/lmsans9oblique"
	"github.com/go-fonts/latin-modern/lmsans9regular"
	"github.com/go-fonts/latin-modern/lmsansdemicond10regular"
	"github.com/go-fonts/latin-modern/lmsansquot8oblique"
	"github.com/go-fonts/latin-modern/lmsansquot8regular"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/font/opentype"
)

const mmPerPt = 25.4 / 72.0

var (
	once       sync.Once
	collection []*font.Face
)

// FontCollection returns a collection of all of the cmr TeX fonts for math.
// This can be used by go-text font shaping.
func FontCollection() []*font.Face {
	once.Do(func() {
		// cmbsy
		register(lmmath.TTF)
		// cmr
		register(lmroman17regular.TTF)
		register(lmroman12regular.TTF)
		register(lmroman10regular.TTF)
		register(lmroman9regular.TTF)
		register(lmroman8regular.TTF)
		register(lmroman7regular.TTF)
		register(lmroman6regular.TTF)
		register(lmroman5regular.TTF)
		// cmb, cmbx
		register(lmroman12bold.TTF)
		register(lmroman10bold.TTF)
		register(lmroman9bold.TTF)
		register(lmroman8bold.TTF)
		register(lmroman7bold.TTF)
		register(lmroman6bold.TTF)
		register(lmroman5bold.TTF)
		// cmti
		register(lmroman12italic.TTF)
		register(lmroman10italic.TTF)
		register(lmroman9italic.TTF)
		register(lmroman8italic.TTF)
		register(lmroman7italic.TTF)
		// cmsl
		register(lmromanslant17regular.TTF)
		register(lmromanslant12regular.TTF)
		register(lmromanslant10regular.TTF)
		register(lmromanslant9regular.TTF)
		register(lmromanslant8regular.TTF)
		// cmbxsl
		register(lmromanslant10bold.TTF)
		// cmbxti, cmmib with cmapCMMI
		register(lmroman10bolditalic.TTF)
		// cmcsc
		register(lmromancaps10regular.TTF)
		// cmdunh
		register(lmromandunh10regular.TTF)
		// cmu
		register(lmromanunsl10regular.TTF)

		// cmss
		register(lmsans17regular.TTF)
		register(lmsans12regular.TTF)
		register(lmsans10regular.TTF)
		register(lmsans9regular.TTF)
		register(lmsans8regular.TTF)
		// cmssb, cmssbx
		register(lmsans10bold.TTF)
		// cmssdc
		register(lmsansdemicond10regular.TTF)
		// cmssi
		register(lmsans17oblique.TTF)
		register(lmsans12oblique.TTF)
		register(lmsans10oblique.TTF)
		register(lmsans9oblique.TTF)
		register(lmsans8oblique.TTF)
		// cmssq
		register(lmsansquot8regular.TTF)
		// cmssqi
		register(lmsansquot8oblique.TTF)

		// cmtt
		register(lmmono12regular.TTF)
		register(lmmono10regular.TTF)
		register(lmmono9regular.TTF)
		register(lmmono8regular.TTF)
		// cmti
		// register(lmmono12italic.TTF)
		register(lmmono10italic.TTF)
		// register(lmmono9italic.TTF)
		// register(lmmono8italic.TTF)
		// cmtcsc
		register(lmmonocaps10regular.TTF)

		// Ensure that any outside appends will not reuse the backing store.
		n := len(collection)
		collection = collection[:n:n]
	})
	return collection
}

func register(ttf []byte) {
	faces, err := font.ParseTTC(bytes.NewReader(ttf))
	if err != nil {
		panic(fmt.Errorf("failed to parse font: %v", err))
	}
	collection = append(collection, faces[0])
}

//////// dviFonts

// dviFonts supports rendering of following standard DVI fonts:
//
//	cmr: Roman (5--10pt)
//	cmmi: Math Italic (5--10pt)
//	cmsy: Math Symbols (5--10pt)
//	cmex: Math Extension (10pt)
//	cmss: Sans serif (10pt)
//	cmssqi: Sans serif quote italic (8pt)
//	cmssi: Sans serif Italic (10pt)
//	cmbx: Bold Extended (10pt)
//	cmtt: Typewriter (8--10pt)
//	cmsltt: Slanted typewriter (10pt)
//	cmsl: Slanted roman (8--10pt)
//	cmti: Text italic (7--10pt)
//	cmu: Unslanted text italic (10pt)
//	cmmib: Bold math italic (10pt)
//	cmbsy: Bold math symbols (10pt)
//	cmcsc: Caps and Small caps (10pt)
//	cmssbx: Sans serif bold extended (10pt)
//	cmdunh: Dunhill style (10pt)
type dviFonts struct {
	font map[string]*dviFont
}

type dviFont struct {
	face   *font.Face
	cmap   map[uint32]rune
	size   float32
	italic bool
}

func newFonts() *dviFonts {
	return &dviFonts{
		font: map[string]*dviFont{},
	}
}

func (fs *dviFonts) Get(name string, scale float32) *dviFont {
	i := 0
	for i < len(name) && 'a' <= name[i] && name[i] <= 'z' {
		i++
	}
	fontname := name[:i]
	fontsize := float32(10.0)
	if ifontsize, err := strconv.Atoi(name[i:]); err == nil {
		fontsize = float32(ifontsize)
	}
	fmt.Println("font name:", fontname, fontsize, scale)

	cmap := cmapCMR
	f, ok := fs.font[name]
	if !ok {
		var fontSizes map[float32][]byte
		switch fontname {
		case "cmb", "cmbx":
			fontSizes = map[float32][]byte{
				12.0: lmroman12bold.TTF,
				10.0: lmroman10bold.TTF,
				9.0:  lmroman9bold.TTF,
				8.0:  lmroman8bold.TTF,
				7.0:  lmroman7bold.TTF,
				6.0:  lmroman6bold.TTF,
				5.0:  lmroman5bold.TTF,
			}
		case "cmbsy":
			cmap = cmapCMSY
			fontSizes = map[float32][]byte{
				fontsize: lmmath.TTF,
			}
		case "cmbxsl":
			fontSizes = map[float32][]byte{
				fontsize: lmromanslant10bold.TTF,
			}
		case "cmbxti":
			fontSizes = map[float32][]byte{
				10.0: lmroman10bolditalic.TTF,
			}
		case "cmcsc":
			cmap = cmapCMTT
			fontSizes = map[float32][]byte{
				10.0: lmromancaps10regular.TTF,
			}
		case "cmdunh":
			fontSizes = map[float32][]byte{
				10.0: lmromandunh10regular.TTF,
			}
		case "cmex":
			cmap = cmapCMEX
			fontSizes = map[float32][]byte{
				fontsize: lmmath.TTF,
			}
		case "cmitt":
			cmap = cmapCMTT
			fontSizes = map[float32][]byte{
				10.0: lmmono10italic.TTF,
			}
		case "cmmi":
			cmap = cmapCMMI
			fontSizes = map[float32][]byte{
				12.0: lmroman12italic.TTF,
				10.0: lmroman10italic.TTF,
				9.0:  lmroman9italic.TTF,
				8.0:  lmroman8italic.TTF,
				7.0:  lmroman7italic.TTF,
			}
		case "cmmib":
			cmap = cmapCMMI
			fontSizes = map[float32][]byte{
				10.0: lmroman10bolditalic.TTF,
			}
		case "cmr":
			fontSizes = map[float32][]byte{
				17.0: lmroman17regular.TTF,
				12.0: lmroman12regular.TTF,
				10.0: lmroman10regular.TTF,
				9.0:  lmroman9regular.TTF,
				8.0:  lmroman8regular.TTF,
				7.0:  lmroman7regular.TTF,
				6.0:  lmroman6regular.TTF,
				5.0:  lmroman5regular.TTF,
			}
		case "cmsl":
			fontSizes = map[float32][]byte{
				17.0: lmromanslant17regular.TTF,
				12.0: lmromanslant12regular.TTF,
				10.0: lmromanslant10regular.TTF,
				9.0:  lmromanslant9regular.TTF,
				8.0:  lmromanslant8regular.TTF,
			}
		case "cmsltt":
			fontSizes = map[float32][]byte{
				10.0: lmmonoslant10regular.TTF,
			}
		case "cmss":
			fontSizes = map[float32][]byte{
				17.0: lmsans17regular.TTF,
				12.0: lmsans12regular.TTF,
				10.0: lmsans10regular.TTF,
				9.0:  lmsans9regular.TTF,
				8.0:  lmsans8regular.TTF,
			}
		case "cmssb", "cmssbx":
			fontSizes = map[float32][]byte{
				10.0: lmsans10bold.TTF,
			}
		case "cmssdc":
			fontSizes = map[float32][]byte{
				10.0: lmsansdemicond10regular.TTF,
			}
		case "cmssi":
			fontSizes = map[float32][]byte{
				17.0: lmsans17oblique.TTF,
				12.0: lmsans12oblique.TTF,
				10.0: lmsans10oblique.TTF,
				9.0:  lmsans9oblique.TTF,
				8.0:  lmsans8oblique.TTF,
			}
		case "cmssq":
			fontSizes = map[float32][]byte{
				8.0: lmsansquot8regular.TTF,
			}
		case "cmssqi":
			fontSizes = map[float32][]byte{
				8.0: lmsansquot8oblique.TTF,
			}
		case "cmsy":
			cmap = cmapCMSY
			fontSizes = map[float32][]byte{
				fontsize: lmmath.TTF,
			}
		case "cmtcsc":
			cmap = cmapCMTT
			fontSizes = map[float32][]byte{
				10.0: lmmonocaps10regular.TTF,
			}
		//case "cmtex":
		//cmap = nil
		case "cmti":
			fontSizes = map[float32][]byte{
				12.0: lmroman12italic.TTF,
				10.0: lmroman10italic.TTF,
				9.0:  lmroman9italic.TTF,
				8.0:  lmroman8italic.TTF,
				7.0:  lmroman7italic.TTF,
			}
		case "cmtt":
			cmap = cmapCMTT
			fontSizes = map[float32][]byte{
				12.0: lmmono12regular.TTF,
				10.0: lmmono10regular.TTF,
				9.0:  lmmono9regular.TTF,
				8.0:  lmmono8regular.TTF,
			}
		case "cmu":
			fontSizes = map[float32][]byte{
				10.0: lmromanunsl10regular.TTF,
			}
		//case "cmvtt":
		//cmap = cmapCTT
		default:
			fmt.Println("WARNING: unknown font", fontname)
		}

		// select closest matching font size
		var data []byte
		var size float32
		for isize, idata := range fontSizes {
			if data == nil || math32.Abs(isize-fontsize) < math32.Abs(size-fontsize) {
				data = idata
				size = isize
			}
		}

		// load font
		faces, err := font.ParseTTC(bytes.NewReader(data))
		if err != nil {
			fmt.Println("ERROR: %w", err)
		}
		face := faces[0]
		fsize := scale * fontsize
		isItalic := 0 < len(fontname) && fontname[len(fontname)-1] == 'i'
		fsizeCorr := float32(1.0)

		f = &dviFont{face, cmap, fsizeCorr * fsize, isItalic}
		fs.font[name] = f
	}
	return f
}

func (f *dviFont) Draw(p *ppath.Path, x, y float32, cid uint32, scale float32) float32 {
	r := f.cmap[cid]
	face := f.face
	gid, ok := face.Cmap.Lookup(r)
	if !ok {
		fmt.Println("rune not found:", string(r))
	}

	outline := face.GlyphData(gid).(font.GlyphOutline)
	sc := scale * f.size / float32(face.Upem())
	// fmt.Println("draw scale:", sc, "f.size:", f.size, "face.Upem()", face.Upem())

	// this random hack fixes the \sum formatting but the source of the problem
	// is in star-tex, not here:
	// ext, ok := face.FontHExtents()
	// fmt.Printf("%#v\n", ext)
	// y += sc * (float32(ext.LineGap) + (1123 - ext.Ascender) + (292 + ext.Descender))

	if f.italic {
		// angle := f.face.Post.ItalicAngle
		// angle := float32(-15) // degrees
		// x -= scale * f.size * face.LineMetric(font.XHeight) / 2.0 * math32.Tan(-angle*math.Pi/180.0)
	}

	for _, s := range outline.Segments {
		p0 := math32.Vec2(s.Args[0].X*sc+x, -s.Args[0].Y*sc+y)
		switch s.Op {
		case opentype.SegmentOpMoveTo:
			p.MoveTo(p0.X, p0.Y)
		case opentype.SegmentOpLineTo:
			p.LineTo(p0.X, p0.Y)
		case opentype.SegmentOpQuadTo:
			p1 := math32.Vec2(s.Args[1].X*sc+x, -s.Args[1].Y*sc+y)
			p.QuadTo(p0.X, p0.Y, p1.X, p1.Y)
		case opentype.SegmentOpCubeTo:
			p1 := math32.Vec2(s.Args[1].X*sc+x, -s.Args[1].Y*sc+y)
			p2 := math32.Vec2(s.Args[2].X*sc+x, -s.Args[2].Y*sc+y)
			p.CubeTo(p0.X, p0.Y, p1.X, p1.Y, p2.X, p2.Y)
		}
	}
	p.Close()
	adv := sc * face.HorizontalAdvance(gid)
	fmt.Println("hadv:", face.HorizontalAdvance(gid), "adv:", adv)
	return adv
}
