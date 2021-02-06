// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"unicode/utf8"
)

// XMLEncoder is a minimal XML encoder that formats output with Attr
// each on a new line, using same API as xml.Encoder
type XMLEncoder struct {
	Writer      io.Writer
	DoIndent    bool
	IndBytes    []byte
	PreBytes    []byte
	CurIndent   int
	CurStart    string
	NoEndIndent bool
}

func NewXMLEncoder(wr io.Writer) *XMLEncoder {
	return &XMLEncoder{Writer: wr}
}

func (xe *XMLEncoder) Indent(prefix, indent string) {
	if len(indent) > 0 {
		xe.DoIndent = true
	}
	xe.IndBytes = []byte(indent)
	xe.PreBytes = []byte(prefix)
}

func (xe *XMLEncoder) EncodeToken(t xml.Token) error {
	switch t := t.(type) {
	case xml.StartElement:
		if err := xe.WriteStart(&t); err != nil {
			return err
		}
	case xml.EndElement:
		if err := xe.WriteEnd(t.Name.Local); err != nil {
			return err
		}
	case xml.CharData:
		if xe.CurStart != "" {
			xe.WriteString(">")
			xe.CurStart = ""
			xe.NoEndIndent = true // don't indent the end now
		}
		EscapeText(xe.Writer, t, false)
	}
	return nil
}

func (xe *XMLEncoder) WriteString(str string) {
	xe.Writer.Write([]byte(str))
}

func (xe *XMLEncoder) WriteIndent() {
	xe.Writer.Write(xe.PreBytes)
	xe.Writer.Write(bytes.Repeat(xe.IndBytes, xe.CurIndent))
}

func (xe *XMLEncoder) WriteEOL() {
	xe.Writer.Write([]byte("\n"))
}

// Decide whether the given rune is in the XML Character Range, per
// the Char production of https://www.xml.com/axml/testaxml.htm,
// Section 2.2 Characters.
func isInCharacterRange(r rune) (inrange bool) {
	return r == 0x09 ||
		r == 0x0A ||
		r == 0x0D ||
		r >= 0x20 && r <= 0xD7FF ||
		r >= 0xE000 && r <= 0xFFFD ||
		r >= 0x10000 && r <= 0x10FFFF
}

var (
	escQuot = []byte("&#34;") // shorter than "&quot;"
	escApos = []byte("&#39;") // shorter than "&apos;"
	escAmp  = []byte("&amp;")
	escLT   = []byte("&lt;")
	escGT   = []byte("&gt;")
	escTab  = []byte("&#x9;")
	escNL   = []byte("&#xA;")
	escCR   = []byte("&#xD;")
	escFFFD = []byte("\uFFFD") // Unicode replacement character
)

// XMLEscapeText writes to w the properly escaped XML equivalent
// of the plain text data s. If escapeNewline is true, newline
// XMLcharacters will be escaped.
func EscapeText(w io.Writer, s []byte, escapeNewline bool) error {
	var esc []byte
	last := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRune(s[i:])
		i += width
		switch r {
		case '"':
			esc = escQuot
		case '\'':
			esc = escApos
		case '&':
			esc = escAmp
		case '<':
			esc = escLT
		case '>':
			esc = escGT
		case '\t':
			esc = escTab
		case '\n':
			if !escapeNewline {
				continue
			}
			esc = escNL
		case '\r':
			esc = escCR
		default:
			if !isInCharacterRange(r) || (r == 0xFFFD && width == 1) {
				esc = escFFFD
				break
			}
			continue
		}
		if _, err := w.Write(s[last : i-width]); err != nil {
			return err
		}
		if _, err := w.Write(esc); err != nil {
			return err
		}
		last = i
	}
	_, err := w.Write(s[last:])
	return err
}

// EscapeString writes to p the properly escaped XML equivalent
// of the plain text data s.
func (xe *XMLEncoder) EscapeString(s string, escapeNewline bool) {
	var esc []byte
	last := 0
	for i := 0; i < len(s); {
		r, width := utf8.DecodeRuneInString(s[i:])
		i += width
		switch r {
		case '"':
			esc = escQuot
		case '\'':
			esc = escApos
		case '&':
			esc = escAmp
		case '<':
			esc = escLT
		case '>':
			esc = escGT
		case '\t':
			esc = escTab
		case '\n':
			if !escapeNewline {
				continue
			}
			esc = escNL
		case '\r':
			esc = escCR
		default:
			if !isInCharacterRange(r) || (r == 0xFFFD && width == 1) {
				esc = escFFFD
				break
			}
			continue
		}
		xe.WriteString(s[last : i-width])
		xe.Writer.Write(esc)
		last = i
	}
	xe.WriteString(s[last:])
}

func (xe *XMLEncoder) WriteStart(start *xml.StartElement) error {
	if start.Name.Local == "" {
		return fmt.Errorf("xml: start tag with no name")
	}
	if xe.CurStart != "" {
		xe.WriteString(">")
		xe.WriteEOL()
	}
	xe.WriteIndent()
	xe.WriteString("<")
	xe.WriteString(start.Name.Local)
	xe.CurIndent++

	xe.CurStart = start.Name.Local

	// Attributes
	for _, attr := range start.Attr {
		name := attr.Name
		if name.Local == "" {
			continue
		}
		xe.WriteEOL()
		xe.WriteIndent()
		xe.WriteString(name.Local)
		xe.WriteString(`="`)
		xe.EscapeString(attr.Value, false)
		xe.WriteString(`"`)
	}
	return nil
}

func (xe *XMLEncoder) WriteEnd(name string) error {
	xe.CurIndent--
	if name == "" {
		return fmt.Errorf("xml: end tag with no name")
	}
	if xe.CurStart == name {
		xe.WriteString(" />")
		xe.WriteEOL()
	} else {
		if !xe.NoEndIndent {
			xe.WriteIndent()
		}
		xe.NoEndIndent = false
		xe.WriteString("</")
		xe.WriteString(name)
		xe.WriteString(">")
		xe.WriteEOL()
	}
	xe.CurStart = ""
	xe.Flush()
	return nil
}

func (xe *XMLEncoder) Flush() {
	if bw, isb := xe.Writer.(*bufio.Writer); isb {
		bw.Flush()
	}
}
