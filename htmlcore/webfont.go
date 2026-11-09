// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"bytes"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"sync"

	"cogentcore.org/core/styles"
	"cogentcore.org/core/text/fonts"
	"cogentcore.org/core/text/rich"
	"github.com/aymerick/douceur/css"
)

// FontDecoder, if set, decodes the data of web fonts in formats that the
// text shapers do not read (as WOFF2) into TrueType or OpenType data, for
// [Context.WebFonts]. Fonts that it cannot decode (with an error) are not
// used.
var FontDecoder func(data []byte) ([]byte, error)

// maxWebFont is the largest size of a web font that is loaded.
const maxWebFont = 8 << 20

// webFontURLs are the URLs of the web fonts that have been loaded (or that
// could not be), as they are loaded once for all pages.
var webFontURLs struct {
	sync.Mutex
	loaded map[string]bool
}

// addFontFace loads the web font of the given declarations of an
// @font-face rule, if [Context.WebFonts]: the first of its sources that
// can be loaded, unless its unicode-range does not include Latin letters
// (as for the subsets of fonts for other scripts, which are not used).
func (c *Context) addFontFace(decls []*css.Declaration) {
	var family, src, urange string
	for _, d := range decls {
		switch strings.ToLower(strings.TrimSpace(d.Property)) {
		case "font-family":
			family = strings.Trim(strings.TrimSpace(d.Value), `"'`)
		case "src":
			src = d.Value
		case "unicode-range":
			urange = d.Value
		}
	}
	if family == "" || src == "" || (urange != "" && !unicodeRangeHas(urange, 'A')) {
		return
	}
	for _, s := range splitTopLevel(src, ',') {
		u, format := fontSource(s)
		if u == "" {
			continue // local() fonts, which are not used
		}
		switch format {
		case "", "woff", "woff2", "truetype", "opentype", "woff-variations", "woff2-variations", "truetype-variations", "opentype-variations":
		default:
			continue // as svg and embedded-opentype
		}
		pu, err := parseRelativeURL(u, c.cssBaseURL())
		if err != nil {
			continue
		}
		if c.loadWebFont(family, pu.String()) {
			return
		}
	}
}

// loadWebFont loads the web font of the given family at the given URL,
// once for all pages, and returns whether it is loaded.
func (c *Context) loadWebFont(family, u string) bool {
	key := strings.ToLower(family) + " " + u
	webFontURLs.Lock()
	if webFontURLs.loaded == nil {
		webFontURLs.loaded = map[string]bool{}
	}
	loaded, tried := webFontURLs.loaded[key]
	webFontURLs.Unlock()
	if tried {
		return loaded
	}
	loaded = c.fetchWebFont(family, u)
	webFontURLs.Lock()
	webFontURLs.loaded[key] = loaded
	webFontURLs.Unlock()
	return loaded
}

// fetchWebFont gets the web font at the given URL and adds it for the
// given family, returning whether it did.
func (c *Context) fetchWebFont(family, u string) bool {
	resp, err := Get(c, u)
	if err != nil {
		slog.Debug("htmlcore: getting web font", "url", u, "err", err)
		return false
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxWebFont+1))
	if err != nil || len(data) > maxWebFont {
		return false
	}
	if bytes.HasPrefix(data, []byte("wOF2")) || !isFontData(data) {
		if FontDecoder == nil {
			return false
		}
		if data, err = FontDecoder(data); err != nil {
			slog.Debug("htmlcore: decoding web font", "url", u, "err", err)
			return false
		}
	}
	if err := fonts.AddWebFont(family, data); err != nil {
		slog.Debug("htmlcore: adding web font", "url", u, "err", err)
		return false
	}
	return true
}

// isFontData returns whether the given data is a font that the text
// shapers read: TrueType, OpenType (and collections of them), or WOFF.
func isFontData(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	switch string(data[:4]) {
	case "\x00\x01\x00\x00", "OTTO", "true", "ttcf", "wOFF":
		return true
	}
	return false
}

// fontSource returns the URL and format of the given source of the src of
// an @font-face rule: url(...) format(...), or "" for local(...).
func fontSource(s string) (url, format string) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(strings.ToLower(s), "url(") {
		return "", ""
	}
	end := strings.Index(s, ")")
	if end < 0 {
		return "", ""
	}
	url = strings.Trim(strings.TrimSpace(s[4:end]), `"'`)
	rest := strings.ToLower(s[end+1:])
	if i := strings.Index(rest, "format("); i >= 0 {
		f := rest[i+7:]
		if j := strings.Index(f, ")"); j >= 0 {
			format = strings.Trim(strings.TrimSpace(f[:j]), `"'`)
		}
	}
	return url, format
}

// unicodeRangeHas returns whether the given unicode-range includes the
// given rune.
func unicodeRangeHas(urange string, r rune) bool {
	for _, part := range strings.Split(urange, ",") {
		part = strings.ToUpper(strings.TrimSpace(part))
		part = strings.TrimPrefix(part, "U+")
		lo, hi := part, part
		if a, b, ok := strings.Cut(part, "-"); ok {
			lo, hi = a, b
		} else if strings.Contains(part, "?") { // as U+4??
			lo, hi = strings.ReplaceAll(part, "?", "0"), strings.ReplaceAll(part, "?", "F")
		}
		l, err1 := strconv.ParseUint(lo, 16, 32)
		h, err2 := strconv.ParseUint(hi, 16, 32)
		if err1 == nil && err2 == nil && uint64(r) >= l && uint64(r) <= h {
			return true
		}
	}
	return false
}

// webFontFamily returns the first of the families of the given value of
// font-family that is a web font that has been loaded, or "".
func webFontFamily(v string) string {
	for _, f := range strings.Split(v, ",") {
		f = strings.Trim(strings.TrimSpace(f), `"'`)
		if f != "" && fonts.HasWebFont(f) {
			return f
		}
	}
	return ""
}

// splitTopLevel splits the given string at the given separator where it is
// not in parentheses or quotes (as a list of sources with url()s).
func splitTopLevel(s string, sep byte) []string {
	var parts []string
	for {
		before, after, found := cutTopLevel(s, sep)
		parts = append(parts, before)
		if !found {
			return parts
		}
		s = after
	}
}

// setFontFamily sets the font family of the given style to that of the
// given value of font-family: the first web font of it that is loaded (as
// the Custom family, with its generic family after it), if
// [Context.WebFonts], or else the given generic family.
func (c *Context) setFontFamily(s *styles.Style, v string, generic rich.Family) {
	s.Font.Family = generic
	if !c.WebFonts {
		return
	}
	if wf := webFontFamily(v); wf != "" {
		s.Font.Family = rich.Custom
		s.Font.CustomFont = rich.FontName(wf + ", " + rich.Settings.Family(generic))
	}
}
