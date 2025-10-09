// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package pdf

import (
	"bytes"
	"compress/zlib"
	"encoding/ascii85"
	"fmt"
	"image"
	"io"
	"math"
	"slices"
	"sort"
	"strings"
	"time"
	"unicode/utf16"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
	"golang.org/x/exp/maps"
)

// TODO: Invalid graphics transparency, Group has a transparency S entry or the S entry is null
// TODO: Invalid Color space, The operator "g" can't be used without Color Profile

type pdfWriter struct {
	w   io.Writer
	err error

	unitContext units.Context
	globalScale float32 // global unit conversion
	pos         int
	objOffsets  []int
	pages       []pdfRef

	page     *pdfPage
	fontsStd map[string]pdfRef
	// todo: for custom fonts:
	// fontSubset map[*text.Font]*ppath.FontSubsetter
	// fontsH   map[*text.Font]pdfRef
	// fontsV   map[*text.Font]pdfRef
	images   map[image.Image]pdfRef
	layers   pdfLayers
	anchors  pdfMap // things that can be linked to within doc
	compress bool
	subset   bool
	title    string
	subject  string
	keywords string
	author   string
	creator  string
	lang     string
}

func newPDFWriter(writer io.Writer, un *units.Context) *pdfWriter {
	w := &pdfWriter{
		w:           writer,
		unitContext: *un,
		objOffsets:  []int{0, 0, 0}, // catalog, metadata, page tree
		fontsStd:    map[string]pdfRef{},
		// fontSubset: map[*text.Font]*ppath.FontSubsetter{},
		// fontsH:   map[*text.Font]pdfRef{},
		// fontsV:   map[*text.Font]pdfRef{},
		images:   map[image.Image]pdfRef{},
		compress: false,
		subset:   true,
	}
	w.layerInit()

	w.globalScale = w.unitContext.Convert(1, units.UnitDot, units.UnitPt)

	w.write("%%PDF-1.7\n%%Ŧǟċơ\n")
	return w
}

// SetCompression enable the compression of the streams.
func (w *pdfWriter) SetCompression(compress bool) {
	w.compress = compress
}

// SeFontSubsetting enables the subsetting of embedded fonts.
func (w *pdfWriter) SetFontSubsetting(subset bool) {
	w.subset = subset
}

// SetTitle sets the document's title.
func (w *pdfWriter) SetTitle(title string) {
	w.title = title
}

// SetSubject sets the document's subject.
func (w *pdfWriter) SetSubject(subject string) {
	w.subject = subject
}

// SetKeywords sets the document's keywords.
func (w *pdfWriter) SetKeywords(keywords string) {
	w.keywords = keywords
}

// SetAuthor sets the document's author.
func (w *pdfWriter) SetAuthor(author string) {
	w.author = author
}

// SetCreator sets the document's creator.
func (w *pdfWriter) SetCreator(creator string) {
	w.creator = creator
}

// SetLang sets the document's language.
func (w *pdfWriter) SetLang(lang string) {
	w.lang = lang
}

func (w *pdfWriter) writeBytes(b []byte) {
	if w.err != nil {
		return
	}
	n, err := w.w.Write(b)
	w.pos += n
	w.err = err
}

func (w *pdfWriter) write(s string, v ...interface{}) {
	if w.err != nil {
		return
	}
	n, err := fmt.Fprintf(w.w, s, v...)
	w.pos += n
	w.err = err
}

type pdfRef int
type pdfName string
type pdfArray []interface{}
type pdfDict map[pdfName]interface{}
type pdfMap map[string]interface{}
type pdfFilter string
type pdfStream struct {
	dict   pdfDict
	stream []byte
}

const (
	pdfFilterASCII85 pdfFilter = "ASCII85Decode"
	pdfFilterFlate   pdfFilter = "FlateDecode"
	pdfFilterDCT     pdfFilter = "DCTDecode"
)

func pdfValContinuesName(val any) bool {
	switch val.(type) {
	case string, pdfName, pdfFilter, pdfArray, pdfDict, pdfStream:
		return false
	}
	return true
}

func (w *pdfWriter) writeVal(i interface{}) {
	switch v := i.(type) {
	case bool:
		if v {
			w.write("true")
		} else {
			w.write("false")
		}
	case int:
		w.write("%d", v)
	case float32:
		w.write("%v", dec(v))
	case float64:
		w.write("%v", dec(v))
	case string:
		w.write("(%v)", escape(v))
	case pdfRef:
		w.write("%v 0 R", v)
	case pdfName, pdfFilter:
		w.write("/%v", v)
	case pdfArray:
		w.write("[")
		for j, val := range v {
			if j != 0 {
				w.write(" ")
			}
			w.writeVal(val)
		}
		w.write("]")
	case pdfDict:
		w.write("<<")
		if val, ok := v["Type"]; ok {
			w.write("/Type")
			if pdfValContinuesName(val) {
				w.write(" ")
			}
			w.writeVal(val)
		}
		if val, ok := v["Subtype"]; ok {
			w.write("/Subtype")
			if pdfValContinuesName(val) {
				w.write(" ")
			}
			w.writeVal(val)
		}
		if val, ok := v["S"]; ok {
			w.write("/S")
			if pdfValContinuesName(val) {
				w.write(" ")
			}
			w.writeVal(val)
		}
		keys := []string{}
		for key := range v {
			if key != "Type" && key != "Subtype" && key != "S" {
				keys = append(keys, string(key))
			}
		}
		sort.Strings(keys)
		for _, key := range keys {
			w.writeVal(pdfName(key))
			if pdfValContinuesName(v[pdfName(key)]) {
				w.write(" ")
			}
			w.writeVal(v[pdfName(key)])
		}
		w.write(">>")
	case pdfMap:
		w.write("<<")
		keys := maps.Keys(v)
		sort.Strings(keys)
		nk := len(keys)
		for i, key := range keys {
			w.writeVal(pdfName(key))
			w.write(" ")
			w.writeVal(v[key])
			if i < nk-1 {
				w.write(" ")
			}
		}
		w.write(">>")
	case pdfStream:
		if v.dict == nil {
			v.dict = pdfDict{}
		}

		filters := []pdfFilter{}
		if filter, ok := v.dict["Filter"].(pdfFilter); ok {
			filters = append(filters, filter)
		} else if filterArray, ok := v.dict["Filter"].(pdfArray); ok {
			for i := len(filterArray) - 1; i >= 0; i-- {
				if filter, ok := filterArray[i].(pdfFilter); ok {
					filters = append(filters, filter)
				}
			}
		}

		b := v.stream
		for _, filter := range filters {
			var b2 bytes.Buffer
			switch filter {
			case pdfFilterASCII85:
				w := ascii85.NewEncoder(&b2)
				w.Write(b)
				w.Close()
				fmt.Fprintf(&b2, "~>")
				b = b2.Bytes()
			case pdfFilterFlate:
				w := zlib.NewWriter(&b2)
				w.Write(b)
				w.Close()
				b = b2.Bytes()
			default:
				// assume already in the right format
			}
		}

		v.dict["Length"] = len(b)
		w.writeVal(v.dict)
		w.write("stream\n")
		w.writeBytes(b)
		w.write("\nendstream\n")
	case *pdfLayer:
		v.objNum = len(w.objOffsets)
		w.write("<</Type /OCG /Name %s>>", pdfName(v.name))
	default:
		// panic(fmt.Sprintf("unknown PDF type %T", i))
	}
}

func (w *pdfWriter) writeObject(val interface{}) pdfRef {
	// newlines before and after obj and endobj are required by PDF/A
	w.objOffsets = append(w.objOffsets, w.pos)
	w.write("%v 0 obj\n", len(w.objOffsets))
	w.writeVal(val)
	w.write("\nendobj\n")
	return pdfRef(len(w.objOffsets))
}

func standardFontName(sty *rich.Style) string {
	name := "Helvetica"
	switch sty.Family {
	case rich.SansSerif:
		name = "Helvetica"
	case rich.Serif:
		name = "Times"
	case rich.Monospace:
		name = "Courier"
	case rich.Cursive:
		name = "ZapfChancery"
	case rich.Math:
		name = "Symbol"
	case rich.Emoji:
		name = "ZapfDingbats"
	}
	if sty.Weight > rich.Medium {
		name += "-Bold"
		if sty.Slant == rich.Italic {
			if name == "Times" {
				name += "Italic"
			} else {
				name += "Oblique"
			}
		}
	} else {
		if sty.Slant == rich.Italic {
			if name == "Times" {
				name += "-Italic"
			} else {
				name += "-Oblique"
			}
		}
	}
	if name == "Times" {
		name = "Times-Roman" // ugh
	}
	return name
}

func (w *pdfWriter) getFont(sty *rich.Style, tsty *text.Style) pdfRef {
	if sty.Family != rich.Custom {
		stdFont := standardFontName(sty)
		if ref, ok := w.fontsStd[stdFont]; ok {
			return ref
		}

		dict := pdfDict{
			"Type":     pdfName("Font"),
			"Subtype":  pdfName("Type1"),
			"BaseFont": pdfName(stdFont),
			"Encoding": pdfName("WinAnsiEncoding"),
		}
		ref := w.writeObject(dict)
		w.fontsStd[stdFont] = ref
		return ref
	}
	// todo: deal with custom
	/*
		fonts := w.fontsH
		if vertical {
			fonts = w.fontsV
		}
		if ref, ok := fonts[font]; ok {
			return ref
		}
		w.objOffsets = append(w.objOffsets, 0)
		ref := pdfRef(len(w.objOffsets))
		fonts[font] = ref
		w.fontSubset[font] = ppath.NewFontSubsetter()
		return ref
	*/
	return 0
}

// Close finished the document.
func (w *pdfWriter) Close() error {
	if w.page != nil {
		w.pages = append(w.pages, w.page.writePage(pdfRef(3)))
	}

	kids := pdfArray{}
	for _, page := range w.pages {
		kids = append(kids, page)
	}

	// write fonts
	// w.writeFonts(w.fontsH, false)
	// w.writeFonts(w.fontsV, false)

	// document catalog
	catalog := pdfDict{
		"Type":  pdfName("Catalog"),
		"Pages": pdfRef(3),
	}

	if len(w.anchors) > 0 {
		ancrefs := pdfMap{}
		var nms []string
		for nm, v := range w.anchors {
			vary := v.(pdfArray)
			vary[0] = w.pages[vary[0].(int)] // replace page no with ref
			anc := w.writeObject(pdfDict{"D": vary})
			ancrefs[nm] = anc
			nms = append(nms, nm)
		}
		slices.Sort(nms)
		var nmary pdfArray
		for _, nm := range nms {
			nmary = append(nmary, nm, ancrefs[nm])
		}
		nmref := w.writeObject(pdfDict{"Names": nmary})
		catalog[pdfName("Names")] = pdfDict{"Dests": nmref}
	}

	// document info
	info := pdfDict{
		"Producer":     "cogentcore/pdf",
		"CreationDate": time.Now().Format("D:20060102150405Z0700"),
	}

	encode := func(s string) string {
		// TODO: make clean
		ascii := true
		for _, r := range s {
			if 0x80 <= r {
				ascii = false
				break
			}
		}
		if ascii {
			return s
		}

		rs := utf16.Encode([]rune(s))
		b := make([]byte, 2+2*len(rs))
		b[0] = 254
		b[1] = 255
		for i, r := range rs {
			b[2+2*i+0] = byte(r >> 8)
			b[2+2*i+1] = byte(r & 0x00FF)
		}
		return string(b)
	}
	if w.title != "" {
		info["Title"] = encode(w.title)
	}
	if w.subject != "" {
		info["Subject"] = encode(w.subject)
	}
	if w.keywords != "" {
		info["Keywords"] = encode(w.keywords)
	}
	if w.author != "" {
		info["Author"] = encode(w.author)
	}
	if w.creator != "" {
		info["Creator"] = encode(w.creator)
	}
	if w.lang != "" {
		catalog["Lang"] = encode(w.creator)
	}

	// document catalog
	w.objOffsets[0] = w.pos
	w.write("%v 0 obj\n", 1)
	w.writeVal(catalog)
	w.writeLayerCatalog()
	w.write("\nendobj\n")

	// document info
	w.objOffsets[1] = w.pos
	w.write("%v 0 obj\n", 2)
	w.writeVal(info)
	w.writeLayerResourceDict()
	w.write("\nendobj\n")

	// page tree
	w.objOffsets[2] = w.pos
	w.write("%v 0 obj\n", 3)
	w.writeVal(pdfDict{
		"Type":  pdfName("Pages"),
		"Kids":  pdfArray(kids),
		"Count": len(kids),
	})
	w.write("\nendobj\n")

	xrefOffset := w.pos
	w.write("xref\n0 %d\n0000000000 65535 f \n", len(w.objOffsets)+1)
	for _, objOffset := range w.objOffsets {
		w.write("%010d 00000 n \n", objOffset)
	}
	w.write("trailer\n")
	w.writeVal(pdfDict{
		"Root": pdfRef(1),
		"Size": len(w.objOffsets) + 1,
		"Info": pdfRef(2),
		// TODO: write document ID
	})
	w.write("\nstartxref\n%v\n%%%%EOF\n", xrefOffset)
	return w.err
}

// NewPage starts a new page.
func (w *pdfWriter) NewPage(width, height float32) *pdfPage {
	if w.page != nil {
		w.pages = append(w.pages, w.page.writePage(pdfRef(3)))
	}

	// for defaults see https://help.adobe.com/pdfl_sdk/15/PDFL_SDK_HTMLHelp/PDFL_SDK_HTMLHelp/API_References/PDFL_API_Reference/PDFEdit_Layer/General.html#_t_PDEGraphicState
	w.page = &pdfPage{
		Buffer:         &bytes.Buffer{},
		pdf:            w,
		width:          width,
		height:         height,
		pageNo:         len(w.pages),
		resources:      pdfDict{},
		graphicsStates: map[float32]pdfName{},
		inTextObject:   false,
		textPosition:   math32.Vector2{},
		textCharSpace:  0.0,
		textRenderMode: 0,
	}
	w.page.stack.Push(newContext(styles.NewPaint(), math32.Identity2()))
	w.page.setTopTransform()
	// fmt.Println("added page:", w.page.pageNo)
	return w.page
}

// setTopTransform sets the current transformation matrix so that
// the top left corner is effectively at 0,0. This is set at the
// start of each page, to align with standard rendering in cogent core.
func (w *pdfPage) setTopTransform() {
	sc := w.pdf.globalScale
	m := math32.Translate2D(0, w.height).Scale(sc, -sc)
	w.SetTransform(m)
}

type dec float32

func (f dec) String() string {
	s := fmt.Sprintf("%.*f", 5, f) // precision
	s = string(ppath.MinifyDecimal([]byte(s), ppath.Precision))
	if dec(math.MaxInt32) < f || f < dec(math.MinInt32) {
		if i := strings.IndexByte(s, '.'); i == -1 {
			s += ".0"
		}
	}
	return s
}

// mat2 returns matrix components as a string
func mat2(m math32.Matrix2) string {
	return fmt.Sprintf("%v %v %v %v %v %v", dec(m.XX), dec(m.XY), dec(m.YX), dec(m.YY), dec(m.X0), dec(m.Y0))
}

// Escape special characters in strings
func escape(s string) string {
	s = strings.Replace(s, "\\", "\\\\", -1)
	s = strings.Replace(s, "(", "\\(", -1)
	s = strings.Replace(s, ")", "\\)", -1)
	s = strings.Replace(s, "\r", "\\r", -1)
	return s
}

// utf8toutf16 converts UTF-8 to UTF-16BE; from http://www.fpdf.org/
func utf8toutf16(s string, withBOM ...bool) string {
	bom := true
	if len(withBOM) > 0 {
		bom = withBOM[0]
	}
	res := make([]byte, 0, 8)
	if bom {
		res = append(res, 0xFE, 0xFF)
	}
	nb := len(s)
	i := 0
	for i < nb {
		c1 := byte(s[i])
		i++
		switch {
		case c1 >= 224:
			// 3-byte character
			c2 := byte(s[i])
			i++
			c3 := byte(s[i])
			i++
			res = append(res, ((c1&0x0F)<<4)+((c2&0x3C)>>2),
				((c2&0x03)<<6)+(c3&0x3F))
		case c1 >= 192:
			// 2-byte character
			c2 := byte(s[i])
			i++
			res = append(res, ((c1 & 0x1C) >> 2),
				((c1&0x03)<<6)+(c2&0x3F))
		default:
			// Single-byte character
			res = append(res, 0, c1)
		}
	}
	return string(res)
}
