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
	Date time.Time

	// Authors are the optional authors of the page.
	Authors []string

	// Draft indicates that the page is a draft and should not be visible on the web.
	Draft bool

	// Categories are the categories that the page belongs to.
	Categories []string
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
	return tomlx.ReadBytes(pg, data)
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
	cpages := pagesByCategory[pg.Name]
	if cpages == nil {
		return nil
	}
	res := []byte{'\n'}
	for _, cpage := range cpages {
		res = append(res, fmt.Sprintf("* [[%s]]\n", cpage.Name)...)
	}
	return res
}
