// Copyright (c) 2020, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package markdown

import (
	"log"
	"strings"

	"goki.dev/pi/v2/complete"
	"goki.dev/pi/v2/langs/bibtex"
	"goki.dev/pi/v2/lex"
	"goki.dev/pi/v2/pi"
)

// CompleteCite does completion on citation
func (ml *MarkdownLang) CompleteCite(fss *pi.FileStates, origStr, str string, pos lex.Pos) (md complete.Matches) {
	bfile, has := fss.MetaData("bibfile")
	if !has {
		return
	}
	bf, err := ml.Bibs.Open(bfile)
	if err != nil {
		return
	}
	md.Seed = str
	for _, be := range bf.BibTex.Entries {
		if strings.HasPrefix(be.CiteName, str) {
			c := complete.Completion{Text: be.CiteName, Label: be.CiteName, Icon: "field"}
			md.Matches = append(md.Matches, c)
		}
	}
	return md
}

// LookupCite does lookup on citation
func (ml *MarkdownLang) LookupCite(fss *pi.FileStates, origStr, str string, pos lex.Pos) (ld complete.Lookup) {
	bfile, has := fss.MetaData("bibfile")
	if !has {
		return
	}
	bf, err := ml.Bibs.Open(bfile)
	if err != nil {
		return
	}
	lkbib := bibtex.NewBibTex()
	for _, be := range bf.BibTex.Entries {
		if strings.HasPrefix(be.CiteName, str) {
			lkbib.Entries = append(lkbib.Entries, be)
		}
	}
	if len(lkbib.Entries) > 0 {
		ld.SetFile(fss.Filename, 0, 0)
		ld.Text = []byte(lkbib.PrettyString())
	}
	return ld
}

// OpenBibfile attempts to find the bibliography file, and load it.
// Sets meta data "bibfile" to resulting file if found, and deletes it if not.
func (ml *MarkdownLang) OpenBibfile(fss *pi.FileStates, pfs *pi.FileState) error {
	bfile := ml.FindBibliography(pfs)
	if bfile == "" {
		fss.DeleteMetaData("bibfile")
		return nil
	}
	_, err := ml.Bibs.Open(bfile)
	if err != nil {
		log.Println(err)
		fss.DeleteMetaData("bibfile")
		return err
	}
	fss.SetMetaData("bibfile", bfile)
	return nil
}

// FindBibliography looks for yaml metadata at top of markdown file
func (ml *MarkdownLang) FindBibliography(pfs *pi.FileState) string {
	nlines := pfs.Src.NLines()
	if nlines < 3 {
		return ""
	}
	fln := string(pfs.Src.Lines[0])
	if fln != "---" {
		return ""
	}
	trg := `bibfile: `
	trgln := len(trg)
	mx := min(nlines, 100)
	for i := 1; i < mx; i++ {
		sln := pfs.Src.Lines[i]
		lstr := string(sln)
		if lstr == "---" {
			return ""
		}
		lnln := len(sln)
		if lnln < trgln {
			continue
		}
		if strings.HasPrefix(lstr, trg) {
			fnm := lstr[trgln:lnln]
			if !strings.HasSuffix(fnm, ".bib") {
				fnm += ".bib"
			}
			return fnm
		}
	}
	return ""
}
