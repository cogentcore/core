// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tex

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/goki/pi/complete"
	"github.com/goki/pi/langs/bibtex"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/pi"
)

// CompleteCite does completion on citation
func (tl *TexLang) CompleteCite(fss *pi.FileStates, origStr, str string, pos lex.Pos) (md complete.Matches) {
	bfile, has := fss.MetaData("bibfile")
	if !has {
		return
	}
	bd := tl.BibData(bfile)
	if bd == nil {
		return
	}
	md.Seed = str
	for _, be := range bd.BibTex.Entries {
		if strings.HasPrefix(be.CiteName, str) {
			c := complete.Completion{Text: be.CiteName, Label: be.CiteName, Icon: "field"}
			md.Matches = append(md.Matches, c)
		}
	}
	return md
}

// LookupCite does lookup on citation
func (tl *TexLang) LookupCite(fss *pi.FileStates, origStr, str string, pos lex.Pos) (ld complete.Lookup) {
	bfile, has := fss.MetaData("bibfile")
	if !has {
		return
	}
	bd := tl.BibData(bfile)
	if bd == nil {
		return
	}
	lkbib := bibtex.NewBibTex()
	for _, be := range bd.BibTex.Entries {
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

// BibData returns the bib data for given bibfile, non-nil if exists
func (tl *TexLang) BibData(fname string) *BibData {
	if tl.Bibs == nil {
		tl.Bibs = make(map[string]*BibData)
	}
	bd, has := tl.Bibs[fname]
	if !has {
		return nil
	}
	return bd
}

// FindBibfile attempts to find the /bibliography file, and load it
// Returns full path to bib file if found and loaded, else "" and false.
// Sets meta data "bibfile" to resulting file if found, and deletes it if not
func (tl *TexLang) FindBibfile(fss *pi.FileStates, pfs *pi.FileState) (string, bool) {
	bfile := tl.FindBibliography(pfs)
	if bfile == "" {
		fss.DeleteMetaData("bibfile")
		return "", false
	}
	// fmt.Printf("bfile: %s\n", bfile)
	st, err := os.Stat(bfile)
	if os.IsNotExist(err) {
		bin := os.Getenv("BIBINPUTS")
		if bin == "" {
			bin = os.Getenv("TEXINPUTS")
		}
		if bin == "" {
			fmt.Printf("bibtex file not found and no BIBINPUTS or TEXINPUTS set: %s\n", bfile)
			return "", false
		}
		pth := filepath.SplitList(bin)
		got := false
		for _, p := range pth {
			bf := filepath.Join(p, bfile)
			st, err = os.Stat(bf)
			if err == nil {
				bfile = bf
				got = true
				break
			}
		}
		if !got {
			return "", false
		}
	}
	bfile, err = filepath.Abs(bfile)
	if err != nil {
		log.Println(err)
		return "", false
	}
	fss.SetMetaData("bibfile", bfile)
	bd := tl.BibData(bfile)
	if bd != nil && bd.BibTex != nil && bd.Mod == st.ModTime() {
		return bfile, true
	}
	f, err := os.Open(bfile)
	if err != nil {
		log.Println(err)
		return "", false
	}
	defer f.Close()
	parsed, err := bibtex.Parse(f)
	if err != nil {
		fmt.Printf("Bibtex bibliography: %s not loaded due to error(s):\n", bfile)
		log.Println(err)
		return "", false
	}
	if bd == nil {
		bd = &BibData{}
		tl.Bibs[bfile] = bd
	}
	bd.File = bfile
	bd.BibTex = parsed
	bd.Mod = st.ModTime()
	fmt.Printf("(re)loaded bibtex bibliography: %s\n", bfile)
	return bfile, true
}

func (tl *TexLang) FindBibliography(pfs *pi.FileState) string {
	nlines := pfs.Src.NLines()
	trg := `\bibliography{`
	trgln := len(trg)
	for i := nlines - 1; i >= 0; i-- {
		sln := pfs.Src.Lines[i]
		lnln := len(sln)
		if lnln == 0 {
			continue
		}
		if sln[0] != '\\' {
			continue
		}
		if lnln > 100 {
			continue
		}
		lstr := string(sln)
		if strings.HasPrefix(lstr, trg) {
			return lstr[trgln:len(sln)-1] + ".bib"
		}
	}
	return ""
}
