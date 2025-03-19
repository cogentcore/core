// Copyright (c) 2020, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package markdown

import (
	"log"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/text/csl"
	"cogentcore.org/core/text/parse"
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/parse/languages/bibtex"
	"cogentcore.org/core/text/textpos"
)

// CompleteCite does completion on citation
func (ml *MarkdownLang) CompleteCite(fss *parse.FileStates, origStr, str string, pos textpos.Pos) (md complete.Matches) {
	bfile, has := fss.MetaData("bibfile")
	if !has {
		return
	}
	if strings.HasSuffix(bfile, ".bib") {
		bf, err := ml.Bibs.Open(bfile)
		if err != nil {
			return
		}
		md.Seed = str
		for _, be := range bf.BibTex.Entries {
			if strings.HasPrefix(be.CiteName, str) {
				c := complete.Completion{Text: be.CiteName, Label: be.CiteName, Icon: icons.Field}
				md.Matches = append(md.Matches, c)
			}
		}
		return md
	}
	bf, err := ml.CSLs.Open(bfile)
	if errors.Log(err) != nil {
		return
	}
	md.Seed = str
	for _, it := range bf.Items.Values {
		if strings.HasPrefix(it.CitationKey, str) {
			c := complete.Completion{Text: it.CitationKey, Label: it.CitationKey, Icon: icons.Field}
			md.Matches = append(md.Matches, c)
		}
	}
	return md
}

// LookupCite does lookup on citation.
func (ml *MarkdownLang) LookupCite(fss *parse.FileStates, origStr, str string, pos textpos.Pos) (ld complete.Lookup) {
	bfile, has := fss.MetaData("bibfile")
	if !has {
		return
	}
	if strings.HasSuffix(bfile, ".bib") {
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
	bf, err := ml.CSLs.Open(bfile)
	if err != nil {
		return
	}
	var items []csl.Item
	for _, be := range bf.Items.Values {
		if strings.HasPrefix(be.CitationKey, str) {
			items = append(items, *be)
		}
	}
	if len(items) > 0 {
		kl := csl.NewKeyList(items)
		ld.SetFile(fss.Filename, 0, 0)
		ld.Text = []byte(kl.PrettyString())
	}
	return ld
}

// OpenBibfile attempts to find the bibliography file, and load it.
// Sets meta data "bibfile" to resulting file if found, and deletes it if not.
func (ml *MarkdownLang) OpenBibfile(fss *parse.FileStates, pfs *parse.FileState) error {
	bfile := ml.FindBibliography(pfs)
	if bfile == "" {
		fss.DeleteMetaData("bibfile")
		return nil
	}
	if strings.HasSuffix(bfile, ".bib") {
		_, err := ml.Bibs.Open(bfile)
		if err != nil {
			log.Println(err)
			fss.DeleteMetaData("bibfile")
			return err
		}
		fss.SetMetaData("bibfile", bfile)
		return nil
	}
	_, err := ml.CSLs.Open(bfile)
	if err != nil {
		log.Println(err)
		fss.DeleteMetaData("bibfile")
		return err
	}
	fss.SetMetaData("bibfile", bfile)
	return nil
}

// FindBibliography looks for yaml metadata at top of markdown file
func (ml *MarkdownLang) FindBibliography(pfs *parse.FileState) string {
	nlines := pfs.Src.NLines()
	if nlines < 3 {
		return ""
	}
	fln := string(pfs.Src.Lines[0])
	if !(fln == "---" || fln == "+++") {
		return ""
	}
	trg := `bibfile`
	trgln := len(trg)
	mx := min(nlines, 100)
	for i := 1; i < mx; i++ {
		sln := pfs.Src.Lines[i]
		lstr := string(sln)
		if lstr == "---" || lstr == "+++" {
			return ""
		}
		lnln := len(sln)
		if lnln < trgln {
			continue
		}
		if strings.HasPrefix(lstr, trg) {
			fnm := lstr[trgln:lnln]
			if fnm[0] == ':' {
				return fnm
			}
			flds := strings.Fields(fnm)
			if len(flds) != 2 || flds[0] != "=" {
				continue
			}
			if flds[1][0] == '"' {
				fnm = flds[1][1 : len(flds[1])-1]
				return fnm
			}
		}
	}
	return ""
}
