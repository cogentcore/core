// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"bufio"
	"io/fs"
	"time"

	"cogentcore.org/core/base/iox/tomlx"
	"cogentcore.org/core/base/strcase"
)

// Page represents the metadata for a single page of content.
type Page struct {

	// FS is the filesystem that the page is stored in.
	FS fs.FS `toml:"-"`

	// Filename is the name of the file in [Page.FS] that the content is stored in.
	Filename string `toml:"-"`

	// Name is the user-friendly name of the page, defaulting to the
	// [strcase.ToSentence] of the [Page.Filename].
	Name string

	// Date is the optional date that the page was published.
	Date time.Time

	// Authors are the optional authors of the page.
	Authors []string

	// Draft indicates that the page is a draft and should not be visible on the web.
	Draft bool

	// Categories are the categories that the page belongs to.
	Categories []string
}

// NewPage makes a new page in the given filesystem with the given filename,
// sets default values, and reads metadata from the front matter of the page file.
func NewPage(fsys fs.FS, filename string) (*Page, error) {
	pg := &Page{FS: fsys, Filename: filename}
	pg.Defaults()
	err := pg.ReadMetadata()
	return pg, err
}

// Defaults sets default values for the page based on its filename.
func (pg *Page) Defaults() {
	pg.Name = strcase.ToSentence(pg.Filename)
}

// ReadMetadata reads the page metadata from the front matter of the page file,
// if there is any.
func (pg *Page) ReadMetadata() error {
	f, err := pg.FS.Open(pg.Filename)
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
		data = append(data, b...)
	}
	return tomlx.ReadBytes(pg, data)
}
