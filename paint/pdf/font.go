// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package pdf

import (
	"cogentcore.org/core/text/rich"
	"cogentcore.org/core/text/text"
)

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
	bname := name
	if sty.Weight > rich.Medium {
		name += "-Bold"
		if sty.Slant == rich.Italic {
			if bname == "Times" {
				name += "Italic"
			} else {
				name += "Oblique"
			}
		}
	} else {
		if sty.Slant == rich.Italic {
			if bname == "Times" {
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

/*
func (w *pdfWriter) writeFont(ref pdfRef, font *text.Font, vertical bool) {
	// subset the font, we only write the used characters to the PDF CMap object to reduce its
	// length. At the end of the function we add a CID to GID mapping to correctly select the
	// right glyphID.
	sfnt := font.SFNT
	glyphIDs := w.fontSubset[font].List() // also when not subsetting, to minimize cmap table
	if w.subset {
		if sfnt.IsCFF && sfnt.CFF != nil {
			sfnt.CFF.SetGlyphNames(nil)
		}
		sfntSubset, err := sfnt.Subset(glyphIDs, canvasFont.SubsetOptions{Tables: canvasFont.KeepPDFTables})
		if err == nil {
			sfnt = sfntSubset
		} else {
			fmt.Println("WARNING: font subsetting failed:", err)
		}
	}
	fontProgram := sfnt.Write()

	// calculate the character widths for the W array and shorten it
	f := 1000.0 / float64(font.SFNT.Head.UnitsPerEm)
	widths := make([]int, len(glyphIDs)+1)
	for subsetGlyphID, glyphID := range glyphIDs {
		widths[subsetGlyphID] = int(f*float64(font.SFNT.GlyphAdvance(glyphID)) + 0.5)
	}
	DW := widths[0]
	W := pdfArray{}
	i, j := 1, 1
	for k, width := range widths {
		if k != 0 && width != widths[j] {
			if 4 < k-j { // at about 5 equal widths, it would be shorter using the other notation format
				if i < j {
					arr := pdfArray{}
					for _, w := range widths[i:j] {
						arr = append(arr, w)
					}
					W = append(W, i, arr)
				}
				if widths[j] != DW {
					W = append(W, j, k-1, widths[j])
				}
				i = k
			}
			j = k
		}
	}
	if i < len(widths) {
		arr := pdfArray{}
		for _, w := range widths[i:] {
			arr = append(arr, w)
		}
		W = append(W, i, arr)
	}

	// create ToUnicode CMap
	var bfRange, bfChar strings.Builder
	var bfRangeCount, bfCharCount int
	startGlyphID := uint16(0)
	startUnicode := uint32('\uFFFD')
	length := uint16(1)
	for subsetGlyphID, glyphID := range glyphIDs[1:] {
		unicode := uint32(font.SFNT.Cmap.ToUnicode(glyphID))
		if 0x010000 <= unicode && unicode <= 0x10FFFF {
			// UTF-16 surrogates
			unicode -= 0x10000
			unicode = (0xD800+(unicode>>10)&0x3FF)<<16 + 0xDC00 + unicode&0x3FF
		}
		if uint16(subsetGlyphID+1) == startGlyphID+length && unicode == startUnicode+uint32(length) {
			length++
		} else {
			if 1 < length {
				fmt.Fprintf(&bfRange, "\n<%04X> <%04X> <%04X>", startGlyphID, startGlyphID+length-1, startUnicode)
				bfRangeCount++
			} else {
				fmt.Fprintf(&bfChar, "\n<%04X> <%04X>", startGlyphID, startUnicode)
				bfCharCount++
			}
			startGlyphID = uint16(subsetGlyphID + 1)
			startUnicode = unicode
			length = 1
		}
	}
	if 1 < length {
		fmt.Fprintf(&bfRange, "\n<%04X> <%04X> <%04X>", startGlyphID, startGlyphID+length-1, startUnicode)
		bfRangeCount++
	} else {
		fmt.Fprintf(&bfChar, "\n<%04X> <%04X>", startGlyphID, startUnicode)
		bfCharCount++
	}

	toUnicode := bytes.Buffer{}
	fmt.Fprintf(&toUnicode, `/CIDInit /ProcSet findresource begin
12 dict begin
begincmap
/CIDSystemInfo <</Registry(Adobe)/Ordering(UCS)/Supplement 0>> def
/CMapName /Adobe-Identity-UCS def
/CMapType 2 def
1 begincodespacerange
<0000> <FFFF> endcodespacerange`)
	if 0 < bfRangeCount {
		fmt.Fprintf(&toUnicode, `
%d beginbfrange%s endbfrange`, bfRangeCount, bfRange.String())
	}
	if 0 < bfCharCount {
		fmt.Fprintf(&toUnicode, `
%d beginbfchar%s endbfchar`, bfCharCount, bfChar.String())
	}
	fmt.Fprintf(&toUnicode, `
endcmap
CMapName currentdict /CMap defineresource pop
end
end`)
	toUnicodeStream := pdfStream{
		dict:   pdfDict{},
		stream: toUnicode.Bytes(),
	}
	if w.compress {
		toUnicodeStream.dict["Filter"] = pdfFilterFlate
	}
	toUnicodeRef := w.writeObject(toUnicodeStream)

	// write font program
	var cidSubtype string
	var fontfileKey pdfName
	var fontfileRef pdfRef
	if font.SFNT.IsTrueType {
		cidSubtype = "CIDFontType2"
		fontfileKey = "FontFile2"
		fontfileRef = w.writeObject(pdfStream{
			dict: pdfDict{
				"Filter": pdfFilterFlate,
			},
			stream: fontProgram,
		})
	} else if font.SFNT.IsCFF {
		cidSubtype = "CIDFontType0"
		fontfileKey = "FontFile3"
		fontfileRef = w.writeObject(pdfStream{
			dict: pdfDict{
				"Subtype": pdfName("OpenType"),
				"Filter":  pdfFilterFlate,
			},
			stream: fontProgram,
		})
	}

	// get name and CID subtype
	name := font.Name()
	if records := font.SFNT.Name.Get(canvasFont.NamePostScript); 0 < len(records) {
		name = records[0].String()
	}
	baseFont := strings.ReplaceAll(name, " ", "")
	if w.subset {
		baseFont = "SUBSET+" + baseFont // TODO: give unique subset name
	}

	encoding := "Identity-H"
	if vertical {
		encoding = "Identity-V"
	}

	// in order to support more than 256 characters, we need to use a CIDFont dictionary which must be inside a Type0 font. Character codes in the stream are glyph IDs, however for subsetted fonts they are the _old_ glyph IDs, which is why we need the CIDToGIDMap
	dict := pdfDict{
		"Type":      pdfName("Font"),
		"Subtype":   pdfName("Type0"),
		"BaseFont":  pdfName(baseFont),
		"Encoding":  pdfName(encoding), // map character codes in the stream to CID with identity encoding, we additionally map CID to GID in the descendant font when subsetting, otherwise that is also identity
		"ToUnicode": toUnicodeRef,
		"DescendantFonts": pdfArray{pdfDict{
			"Type":     pdfName("Font"),
			"Subtype":  pdfName(cidSubtype),
			"BaseFont": pdfName(baseFont),
			"DW":       DW,
			"W":        W,
			//"CIDToGIDMap": pdfName("Identity"),
			"CIDSystemInfo": pdfDict{
				"Registry":   "Adobe",
				"Ordering":   "Identity",
				"Supplement": 0,
			},
			"FontDescriptor": pdfDict{
				"Type":     pdfName("FontDescriptor"),
				"FontName": pdfName(baseFont),
				"Flags":    4, // Symbolic
				"FontBBox": pdfArray{
					int(f * float64(font.SFNT.Head.XMin)),
					int(f * float64(font.SFNT.Head.YMin)),
					int(f * float64(font.SFNT.Head.XMax)),
					int(f * float64(font.SFNT.Head.YMax)),
				},
				"ItalicAngle": float64(font.SFNT.Post.ItalicAngle),
				"Ascent":      int(f * float64(font.SFNT.Hhea.Ascender)),
				"Descent":     -int(f * float64(font.SFNT.Hhea.Descender)),
				"CapHeight":   int(f * float64(font.SFNT.OS2.SCapHeight)),
				"StemV":       80, // taken from Inkscape, should be calculated somehow, maybe use: 10+220*(usWeightClass-50)/900
				fontfileKey:   fontfileRef,
			},
		}},
	}

	if !w.subset {
		cidToGIDMap := make([]byte, 2*len(glyphIDs))
		for subsetGlyphID, glyphID := range glyphIDs {
			j := int(subsetGlyphID) * 2
			cidToGIDMap[j+0] = byte((glyphID & 0xFF00) >> 8)
			cidToGIDMap[j+1] = byte(glyphID & 0x00FF)
		}
		cidToGIDMapStream := pdfStream{
			dict:   pdfDict{},
			stream: cidToGIDMap,
		}
		if w.compress {
			cidToGIDMapStream.dict["Filter"] = pdfFilterFlate
		}
		cidToGIDMapRef := w.writeObject(cidToGIDMapStream)
		dict["DescendantFonts"].(pdfArray)[0].(pdfDict)["CIDToGIDMap"] = cidToGIDMapRef
	}

	w.objOffsets[ref-1] = w.pos
	w.write("%v 0 obj\n", ref)
	w.writeVal(dict)
	w.write("\nendobj\n")
}

func (w *pdfWriter) writeFonts(fontMap map[*text.Font]pdfRef, vertical bool) {
	// sort fonts by ref to make PDF deterministic
	refs := make([]pdfRef, 0, len(fontMap))
	refMap := make(map[pdfRef]*text.Font, len(fontMap))
	for font, ref := range fontMap {
		refs = append(refs, ref)
		refMap[ref] = font
	}
	sort.Slice(refs, func(i, j int) bool {
		return refs[i] < refs[j]
	})
	for _, ref := range refs {
		w.writeFont(ref, refMap[ref], vertical)
	}
}
*/
