// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"bytes"
	"log"
	"strings"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
	"github.com/goki/gi"
	"github.com/goki/ki"
)

// HiStyleName is a highlighting style name
type HiStyleName string

// HiMarkup manages the syntax highlighting state for TextBuf
type HiMarkup struct {
	Lang      string        `desc:"language for syntax highlighting the code"`
	Style     HiStyleName   `desc:"syntax highlighting style"`
	TabSize   int           `desc:"tab size, in chars"`
	CSSheet   gi.StyleSheet `json:"-" xml:"-" desc:"CSS StyleSheet for given highlighting style"`
	CSSProps  ki.Props      `json:"-" xml:"-" desc:"Commpiled CSS properties for given highlighting style"`
	lastLang  string
	lastStyle HiStyleName
	lexer     chroma.Lexer
	formatter *html.Formatter
	style     *chroma.Style
}

// HasHi returns true if there are highighting parameters set
func (hm *HiMarkup) HasHi() bool {
	if hm.Lang == "" || hm.Style == "" {
		return false
	}
	return true
}

// Init initializes the syntax highlighting for current params
func (hm *HiMarkup) Init() {
	if !hm.HasHi() {
		return
	}
	if hm.Lang == hm.lastLang && hm.Style == hm.lastStyle {
		return
	}
	hm.lexer = chroma.Coalesce(lexers.Get(hm.Lang))
	hm.formatter = html.New(html.WithClasses(), html.TabWidth(hm.TabSize))
	hm.style = styles.Get(string(hm.Style))
	if hm.style == nil {
		hm.style = styles.Fallback
	}
	var cssBuf bytes.Buffer
	err := hm.formatter.WriteCSS(&cssBuf, hm.style)
	if err != nil {
		log.Println(err)
		return
	}
	csstr := cssBuf.String()
	csstr = strings.Replace(csstr, " .chroma .", " .", -1)
	hm.CSSheet.ParseString(csstr)
	hm.CSSProps = hm.CSSheet.CSSProps()

	hm.lastLang = hm.Lang
	hm.lastStyle = hm.Style
}

// MarkupText returns a line-wise split of marked-up text according to current
// syntax highlighting settings
func (hm *HiMarkup) MarkupText(txt []byte) ([][]byte, error) {
	var htmlBuf bytes.Buffer
	iterator, err := hm.lexer.Tokenise(nil, string(txt)) // todo: unfortunate conversion
	err = hm.formatter.Format(&htmlBuf, hm.style, iterator)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	mtlns := bytes.Split(htmlBuf.Bytes(), []byte("\n"))
	return mtlns, nil
}

// MarkupLine returns a marked-up version of line of text
func (hm *HiMarkup) MarkupLine(txtln []byte) ([]byte, error) {
	var htmlBuf bytes.Buffer
	iterator, err := hm.lexer.Tokenise(nil, string(txtln)+"\n")
	// adding \n b/c it needs to see that for comments..
	err = hm.formatter.Format(&htmlBuf, hm.style, iterator)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	b := htmlBuf.Bytes()
	lfidx := bytes.Index(b, []byte("\n"))
	if lfidx > 0 {
		b = b[:lfidx]
	}
	return hm.FixMarkupLine(b), nil
}

// FixMarkupLine fixes the output of chroma markup
func (hm *HiMarkup) FixMarkupLine(mt []byte) []byte {
	mt = bytes.TrimPrefix(mt, []byte(`</span>`)) // leftovers
	mt = bytes.TrimPrefix(mt, []byte(`<span class="err">`))
	mt = bytes.Replace(mt, []byte(`**</span>**`), []byte(`**</span>`), -1)
	mt = bytes.Replace(mt, []byte(`__</span>__`), []byte(`__</span>`), -1)
	return mt
}

// todo: currently based on https://github.com/alecthomas/chroma styles, but we should
// impl our own structured style obj with a list of categories and
// corresponding colors, once we do the parsing etc

// HiStyles are all the available highlighting styles
var HiStyles = []string{
	"abap",
	"algol",
	"algol_nu",
	"api",
	"arduino",
	"autumn",
	"borland",
	"bw",
	"colorful",
	"dracula",
	"emacs",
	"friendly",
	"fruity",
	"github",
	"igor",
	"lovelace",
	"manni",
	"monokai",
	"monokailight",
	"murphy",
	"native",
	"paraiso-dark",
	"paraiso-light",
	"pastie",
	"perldoc",
	"pygments",
	"rainbow_dash",
	"rrt",
	"solarized-dark",
	"solarized-dark256",
	"solarized-light",
	"swapoff",
	"tango",
	"trac",
	"vim",
	"vs",
	"xcode",
}
