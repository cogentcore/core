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
	"github.com/goki/pi/lex"
	"github.com/goki/pi/pi"
	"github.com/nickng/bibtex"
)

// CompleteCite does completion on citation
func (tl *TexLang) CompleteCite(fss *pi.FileStates, origStr, str string, pos lex.Pos) (md complete.Matches) {
	if tl.BibFile == nil {
		return
	}
	md.Seed = str
	for _, be := range tl.BibFile.Entries {
		if strings.HasPrefix(be.CiteName, str) {
			c := complete.Completion{Text: be.CiteName, Label: be.CiteName, Icon: "field"}
			md.Matches = append(md.Matches, c)
		}
	}
	return md
}

// LookupCite does lookup on citation
func (tl *TexLang) LookupCite(fss *pi.FileStates, origStr, str string, pos lex.Pos) (ld complete.Lookup) {
	if tl.BibFile == nil {
		return
	}
	lkbib := bibtex.NewBibTex()
	for _, be := range tl.BibFile.Entries {
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

// FindBibfile attempts to find the /bibliography file, and load it
func (tl *TexLang) FindBibfile(pfs *pi.FileState) {
	bfile := tl.FindBibliography(pfs)
	if bfile == "" {
		return
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
			return
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
			return
		}
	}
	if tl.BibFile != nil && tl.BibFileMod == st.ModTime() {
		return // already up-to-date
	}
	f, err := os.Open(bfile)
	if err != nil {
		log.Println(err)
		return
	}
	defer f.Close()
	parsed, err := bibtex.Parse(f)
	if err != nil {
		fmt.Printf("Bibtex bibliography: %s not loaded due to error(s):\n", bfile)
		log.Println(err)
		return
	}
	tl.BibFile = parsed
	tl.BibFileMod = st.ModTime()
	fmt.Printf("(re)loaded bibtex bibliography: %s\n", bfile)
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
