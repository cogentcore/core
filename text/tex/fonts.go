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

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/text/fonts"
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

// LMFontsLoad loads the LMFonts.
func LMFontsLoad() {
	for i := range LMFonts {
		fd := &LMFonts[i]
		errors.Log(fd.Load())
	}
}

// LMFonts are tex latin-modern fonts.
var LMFonts = []fonts.FontData{
	{Family: "cmbsy", Data: lmmath.TTF},
	{Family: "cmr17", Data: lmroman17regular.TTF},
	{Family: "cmr12", Data: lmroman12regular.TTF},
	{Family: "cmr10", Data: lmroman10regular.TTF},
	{Family: "cmr9", Data: lmroman9regular.TTF},
	{Family: "cmr8", Data: lmroman8regular.TTF},
	{Family: "cmr7", Data: lmroman7regular.TTF},
	{Family: "cmr6", Data: lmroman6regular.TTF},
	{Family: "cmr5", Data: lmroman5regular.TTF},
	// cmb, cmbx
	{Family: "cmb12", Data: lmroman12bold.TTF},
	{Family: "cmb10", Data: lmroman10bold.TTF},
	{Family: "cmb9", Data: lmroman9bold.TTF},
	{Family: "cmb8", Data: lmroman8bold.TTF},
	{Family: "cmb7", Data: lmroman7bold.TTF},
	{Family: "cmb6", Data: lmroman6bold.TTF},
	{Family: "cmb5", Data: lmroman5bold.TTF},
	// cmti
	{Family: "cmti12", Data: lmroman12italic.TTF},
	{Family: "cmti10", Data: lmroman10italic.TTF},
	{Family: "cmti9", Data: lmroman9italic.TTF},
	{Family: "cmti8", Data: lmroman8italic.TTF},
	{Family: "cmti7", Data: lmroman7italic.TTF},
	// cmsl
	{Family: "cmsl17", Data: lmromanslant17regular.TTF},
	{Family: "cmsl12", Data: lmromanslant12regular.TTF},
	{Family: "cmsl10", Data: lmromanslant10regular.TTF},
	{Family: "cmsl9", Data: lmromanslant9regular.TTF},
	{Family: "cmsl8", Data: lmromanslant8regular.TTF},
	// cmbxsl
	{Family: "cmbxsl10", Data: lmromanslant10bold.TTF},
	// cmbxti, cmmib with cmapCMMI
	{Family: "cmmib10", Data: lmroman10bolditalic.TTF},
	// cmcsc
	{Family: "cmcsc10", Data: lmromancaps10regular.TTF},
	// cmdunh
	{Family: "cmdunh10", Data: lmromandunh10regular.TTF},
	// cmu
	{Family: "cmu10", Data: lmromanunsl10regular.TTF},

	// cmss
	{Family: "cmss17", Data: lmsans17regular.TTF},
	{Family: "cmss12", Data: lmsans12regular.TTF},
	{Family: "cmss10", Data: lmsans10regular.TTF},
	{Family: "cmss9", Data: lmsans9regular.TTF},
	{Family: "cmss8", Data: lmsans8regular.TTF},
	// cmssb, cmssbx
	{Family: "cmssb10", Data: lmsans10bold.TTF},
	// cmssdc
	{Family: "cmssdc10", Data: lmsansdemicond10regular.TTF},
	// cmssi
	{Family: "cmssi17", Data: lmsans17oblique.TTF},
	{Family: "cmssi12", Data: lmsans12oblique.TTF},
	{Family: "cmssi10", Data: lmsans10oblique.TTF},
	{Family: "cmssi9", Data: lmsans9oblique.TTF},
	{Family: "cmssi8", Data: lmsans8oblique.TTF},
	// cmssq
	{Family: "cmssq8", Data: lmsansquot8regular.TTF},
	// cmssqi
	{Family: "cmssqi8", Data: lmsansquot8oblique.TTF},

	// cmtt
	{Family: "cmtt12", Data: lmmono12regular.TTF},
	{Family: "cmtt10", Data: lmmono10regular.TTF},
	{Family: "cmtt9", Data: lmmono9regular.TTF},
	{Family: "cmtt8", Data: lmmono8regular.TTF},
	// cmti
	// {Family: "cmti", Data: lmmono12italic.TTF},
	{Family: "cmti10", Data: lmmono10italic.TTF},
	// {Family: "cmti", Data: lmmono9italic.TTF},
	// {Family: "cmti", Data: lmmono8italic.TTF},
	// cmtcsc
	{Family: "cmtcsc10", Data: lmmonocaps10regular.TTF},
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
	font     map[string]*dviFont
	mathSyms *dviFont // always available as backup for any rune
}

type dviFont struct {
	face     *font.Face
	cmap     map[uint32]rune
	size     float32
	italic   bool
	ex       bool
	mathSyms *dviFont // always available as backup for any rune
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
	// fmt.Println("font name:", fontname, fontsize, scale)

	if fs.mathSyms == nil {
		fs.mathSyms = fs.loadFont("cmsy", cmapCMSY, 10.0, scale, lmmath.TTF)
	}

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
		f = fs.loadFont(fontname, cmap, fontsize, scale, data)
		fs.font[name] = f
	}
	return f
}

func (fs *dviFonts) loadFont(fontname string, cmap map[uint32]rune, fontsize, scale float32, data []byte) *dviFont {
	faces, err := font.ParseTTC(bytes.NewReader(data))
	if err != nil { // todo: should still work presumably?
		errors.Log(err)
	}
	face := faces[0]
	fsize := scale * fontsize
	isItalic := 0 < len(fontname) && fontname[len(fontname)-1] == 'i'
	isEx := fontname == "cmex"

	return &dviFont{face: face, cmap: cmap, size: fsize, italic: isItalic, ex: isEx, mathSyms: fs.mathSyms}
}

const (
	mag1 = 1.2
	mag2 = 1.2 * 1.2
	mag3 = 1.2 * 1.2 * 1.2
	mag4 = 1.2 * 1.2 * 1.2 * 1.2 * 1.2
	mag5 = 3.2
)

var cmexScales = map[uint32]float32{
	0x00: mag1,
	0x01: mag1,
	0x02: mag1,
	0x03: mag1,
	0x04: mag1,
	0x05: mag1,
	0x06: mag1,
	0x07: mag1,
	0x08: mag1,
	0x0A: mag1,
	0x0B: mag1,
	0x0C: mag1,
	0x0D: mag1,
	0x0E: mag1,
	0x0F: mag1,

	0x10: mag3, // (
	0x11: mag3, // )
	0x12: mag4, // (
	0x13: mag4, // )
	0x14: mag4, // [
	0x15: mag4, // ]
	0x16: mag4, // ⌊
	0x17: mag4, // ⌋
	0x18: mag4, // ⌈
	0x19: mag4, // ⌉
	0x1A: mag4, // {
	0x1B: mag4, // }
	0x1C: mag4, // 〈
	0x1D: mag4, // 〉
	0x1E: mag4, // ∕
	0x1F: mag4, // \

	0x20: mag5, // (
	0x21: mag5, // )
	0x22: mag5, // [
	0x23: mag5, // ]
	0x24: mag5, // ⌊
	0x25: mag5, // ⌋
	0x26: mag5, // ⌈
	0x27: mag5, // ⌉
	0x28: mag5, // {
	0x29: mag5, // }
	0x2A: mag5, // 〈
	0x2B: mag5, // 〉
	0x2C: mag5, // ∕
	0x2D: mag5, // \
	0x2E: mag3, // ∕
	0x2F: mag3, // \

	0x30: mag2, // ⎛
	0x31: mag2, // ⎞
	0x32: mag2, // ⌈
	0x33: mag2, // ⌉
	0x34: mag2, // ⌊
	0x35: mag2, // ⌋
	0x36: mag2, // ⎢
	0x37: mag2, // ⎥

	0x38: mag2, // ⎧ // big braces start
	0x39: mag2, // ⎫
	0x3A: mag2, // ⎩
	0x3B: mag2, // ⎭
	0x3C: mag2, // ⎨
	0x3D: mag2, // ⎬
	0x3E: mag2, // ⎪
	0x3F: mag2, // ∣ ?? unclear

	0x40: mag2, // ⎝ // big parens
	0x41: mag2, // ⎠
	0x42: mag2, // ⎜
	0x43: mag2, // ⎟
	0x44: mag2, // 〈
	0x45: mag2, // 〉
	0x47: mag2, // ⨆
	0x49: mag2, // ∮
	0x4B: mag2, // ⨀
	0x4D: mag2, // ⨁
	0x4F: mag2, // ⨂

	0x58: mag2, // ∑
	0x59: mag2, // ∏
	0x5A: mag2, // ∫
	0x5B: mag2, // ⋃
	0x5C: mag2, // ⋂
	0x5D: mag2, // ⨄
	0x5E: mag2, // ⋀
	0x5F: mag2, // ⋁

	0x61: mag2, // ∐
	0x63: mag2, // ̂
	0x64: mag4, // ̂
	0x66: mag2, // ˜
	0x67: mag4, // ˜
	0x68: mag3, // [
	0x69: mag3, // ]
	0x6B: mag2, // ⌋
	0x6C: mag2, // ⌈
	0x6D: mag2, // ⌉
	0x6E: mag3, // {
	0x6F: mag3, // }

	0x71: mag3, // √
	0x72: mag4, // √
	0x73: mag5, // √
	0x74: mag1, // ⎷
	0x75: mag1, // ⏐
	0x76: mag1, // ⌜

}

func (f *dviFont) Draw(p *ppath.Path, x, y float32, cid uint32, scale float32) float32 {
	r := f.cmap[cid]
	face := f.face
	gid, ok := face.Cmap.Lookup(r)
	if !ok {
		if f.mathSyms != nil {
			face = f.mathSyms.face
			gid, ok = face.Cmap.Lookup(r)
			if !ok {
				fmt.Println("rune not found in mathSyms:", string(r))
			}
		} else {
			fmt.Println("rune not found:", string(r))
		}
	}
	hadv := face.HorizontalAdvance(gid)
	// fmt.Printf("rune: 0x%0x gid: %d, r: 0x%0X\n", cid, gid, int(r))

	outline := face.GlyphData(gid).(font.GlyphOutline)
	sc := scale * f.size / float32(face.Upem())
	xsc := float32(1)
	// fmt.Println("draw scale:", sc, "f.size:", f.size, "face.Upem()", face.Upem())

	if f.ex {
		ext, _ := face.GlyphExtents(gid)
		exsc, has := cmexScales[cid]
		yb := ext.YBearing
		if has {
			sc *= exsc
			switch cid {
			case 0x5A, 0x49: // \int and \oint are off in large size
				yb += 200
			case 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F:
				// larger delims are too thick
				xsc = .7
			case 0x20, 0x21, 0x22, 0x23, 0x28, 0x29, 0x2A, 0x2B:
				// same for even larger ones
				xsc = .6
			case 0x3C, 0x3D: // braces middles need shifting
				yb += 150
			case 0x3A, 0x3B: // braces bottom shifting
				yb += 400
			// below are fixes for all the square root elements
			case 0x71:
				x += sc * 80
				xsc = .6
			case 0x72:
				x -= sc * 80
				xsc = .6
			case 0x73:
				x -= sc * 80
				xsc = .5
			case 0x74:
				yb += 600
			case 0x75:
				x += sc * 560
			case 0x76:
				x += sc * 400
				yb -= 36
			}
		}
		y += sc * yb
	}

	if f.italic {
		// angle := f.face.Post.ItalicAngle
		// angle := float32(-15) // degrees
		// x -= scale * f.size * face.LineMetric(font.XHeight) / 2.0 * math32.Tan(-angle*math.Pi/180.0)
	}

	for _, s := range outline.Segments {
		p0 := math32.Vec2(s.Args[0].X*xsc*sc+x, -s.Args[0].Y*sc+y)
		switch s.Op {
		case opentype.SegmentOpMoveTo:
			p.MoveTo(p0.X, p0.Y)
		case opentype.SegmentOpLineTo:
			p.LineTo(p0.X, p0.Y)
		case opentype.SegmentOpQuadTo:
			p1 := math32.Vec2(s.Args[1].X*xsc*sc+x, -s.Args[1].Y*sc+y)
			p.QuadTo(p0.X, p0.Y, p1.X, p1.Y)
		case opentype.SegmentOpCubeTo:
			p1 := math32.Vec2(s.Args[1].X*xsc*sc+x, -s.Args[1].Y*sc+y)
			p2 := math32.Vec2(s.Args[2].X*xsc*sc+x, -s.Args[2].Y*sc+y)
			p.CubeTo(p0.X, p0.Y, p1.X, p1.Y, p2.X, p2.Y)
		}
	}
	p.Close()
	adv := sc * hadv
	// fmt.Println("hadv:", face.HorizontalAdvance(gid), "adv:", adv)
	return adv
}
