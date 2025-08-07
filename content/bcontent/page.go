// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bcontent ("base content") provides base types and functions
// shared by both content and the core build tool for content. This is
// necessary to ensure that the core build tool does not import GUI packages.
package bcontent

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"cogentcore.org/core/base/iox/tomlx"
	"cogentcore.org/core/base/strcase"
)

// Page represents the metadata for a single page of content.
type Page struct {

	// Source is the filesystem that the page is stored in.
	Source fs.FS `toml:"-" json:"-"`

	// Filename is the name of the file in [Page.FS] that the content is stored in.
	Filename string `toml:"-" json:"-"`

	// Name is the user-friendly name of the page, defaulting to the
	// [strcase.ToSentence] of the [Page.Filename] without its extension.
	Name string

	// URL is the URL of the page relative to the root of the app, without
	// any leading slash. It defaults to [Page.Name] in kebab-case
	// (ex: "home" or "text-fields"). A blank URL ("") manually
	// specified in the front matter indicates that this the root page.
	URL string

	// Title is the title displayed at the top of the page. It defaults to [Page.Name].
	// Note that [Page.Name] is still used for the stage title and other such things; this
	// is only for the actual title widget.
	Title string

	// Date is the optional date that the page was published.
	Date time.Time `toml:"-"`

	// DateString is only used for parsing the date from the TOML front matter.
	DateString string `toml:"Date" json:"-"`

	// Authors are the optional authors of the page.
	Authors []string

	// Draft indicates that the page is a draft and should not be visible on the web.
	Draft bool

	// Categories are the categories that the page belongs to.
	Categories []string

	// Specials are special content elements for each page
	// that have names with an underscore-delimited key name,
	// such as figure_, table_, sim_ etc, and can be referred
	// to using the #id component of a wikilink. They are rendered
	// using the index of each such element (e.g., Figure 1) in the link.
	Specials map[string][]string
}

// PreRenderPage contains the data for each page printed in JSON by a content app
// run with the generatehtml tag, which is then handled by the core
// build tool.
type PreRenderPage struct {
	Page

	// Description is the automatic page description.
	Description string

	// HTML is the pre-rendered HTML for the page.
	HTML string
}

// NewPage makes a new page in the given filesystem with the given filename,
// sets default values, and reads metadata from the front matter of the page file.
func NewPage(source fs.FS, filename string) (*Page, error) {
	pg := &Page{Source: source, Filename: filename}
	pg.Defaults()
	err := pg.ReadMetadata()
	return pg, err
}

// Defaults sets default values for the page based on its filename.
func (pg *Page) Defaults() {
	pg.Name = strcase.ToSentence(strings.TrimSuffix(pg.Filename, filepath.Ext(pg.Filename)))
	pg.URL = strcase.ToKebab(pg.Name)
	pg.Title = pg.Name
	pg.Categories = []string{"Other"}
}

// ReadMetadata reads the page metadata from the front matter of the page file,
// if there is any.
func (pg *Page) ReadMetadata() error {
	f, err := pg.Source.Open(pg.Filename)
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	var data []byte
	for sc.Scan() {
		b := sc.Bytes()
		if data == nil {
			if string(b) != `+++` {
				return nil
			}
			data = []byte{}
			continue
		}
		if string(b) == `+++` {
			break
		}
		data = append(data, append(b, '\n')...)
	}
	err = tomlx.ReadBytes(pg, data)
	if err != nil {
		return err
	}
	if pg.DateString != "" {
		pg.Date, err = time.Parse(time.DateOnly, pg.DateString)
		if err != nil {
			return err
		}
	}
	return nil
}

// ReadContent returns the page content with any front matter removed.
// It also applies [Page.categoryLinks].
func (pg *Page) ReadContent(pagesByCategory map[string][]*Page) ([]byte, error) {
	b, err := fs.ReadFile(pg.Source, pg.Filename)
	if err != nil {
		return nil, err
	}
	b = append(b, pg.categoryLinks(pagesByCategory)...)
	if !bytes.HasPrefix(b, []byte(`+++`)) {
		return b, nil
	}
	b = bytes.TrimPrefix(b, []byte(`+++`))
	_, after, has := bytes.Cut(b, []byte(`+++`))
	if !has {
		return nil, fmt.Errorf("unclosed front matter")
	}
	return after, nil
}

// categoryLinks, if the page has the same names as one of the given categories,
// returns markdown containing a list of links to all pages in that category.
// Otherwise, it returns nil.
func (pg *Page) categoryLinks(pagesByCategory map[string][]*Page) []byte {
	if pagesByCategory == nil {
		return nil
	}
	cpages := pagesByCategory[pg.Name]
	if cpages == nil {
		return nil
	}
	res := []byte{'\n'}
	for _, cpage := range cpages {
		if cpage == pg {
			continue
		}
		res = append(res, fmt.Sprintf("* [[%s]]\n", cpage.Name)...)
	}
	return res
}

// SpecialName extracts a special element type name from given element name,
// defined as the part before the first underscore _ character.
func SpecialName(name string) string {
	usi := strings.Index(name, "_")
	if usi < 0 {
		return ""
	}
	return name[:usi]
}

// SpecialToKebab does strcase.ToKebab on parts after specialName if present.
func SpecialToKebab(name string) string {
	usi := strings.Index(name, "_")
	if usi < 0 {
		return strcase.ToKebab(name)
	}
	spec := name[:usi+1]
	name = name[usi+1:]
	colon := strings.Index(name, ":")
	if colon > 0 {
		return spec + strcase.ToKebab(name[:colon]) + name[colon:]
	} else {
		return spec + strcase.ToKebab(name)
	}
}

// SpecialLabel returns the label for given special element, using
// the index of the element in the list of specials, e.g., "Figure 1"
func (pg *Page) SpecialLabel(name string) string {
	snm := SpecialName(name)
	if snm == "" {
		return ""
	}
	if pg.Specials == nil {
		b, err := pg.ReadContent(nil)
		if err != nil {
			return ""
		}
		pg.ParseSpecials(b)
	}
	sl := pg.Specials[snm]
	if sl == nil {
		return ""
	}
	i := slices.Index(sl, name)
	if i < 0 {
		return ""
	}
	return strcase.ToSentence(snm) + " " + strconv.Itoa(i+1)
}

// ParseSpecials manually parses specials before rendering md
// because they are needed in advance of generating from md file,
// e.g., for wikilinks.
func (pg *Page) ParseSpecials(b []byte) {
	if pg.Specials != nil {
		return
	}
	pg.Specials = make(map[string][]string)
	scan := bufio.NewScanner(bytes.NewReader(b))
	idt := []byte(`{id="`)
	idn := len(idt)
	for scan.Scan() {
		ln := scan.Bytes()
		n := len(ln)
		if n < idn+1 {
			continue
		}
		if !bytes.HasPrefix(ln, idt) {
			continue
		}
		fs := bytes.Fields(ln) // multiple attributes possible
		ln = fs[0]             // only deal with first one
		id := bytes.TrimSpace(ln[idn:])
		n = len(id)
		if n < 2 {
			continue
		}
		ed := n - 1 // quotes
		if len(fs) == 1 {
			ed = n - 2 // brace
		}
		id = id[:ed]
		sid := string(id)
		snm := SpecialName(sid)
		if snm == "" {
			continue
		}
		// fmt.Println("id:", snm, sid)
		sl := pg.Specials[snm]
		sl = append(sl, sid)
		pg.Specials[snm] = sl
	}
}
