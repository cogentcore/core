// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
	Package spell provides functions for spell check and correction
*/

package spell

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gospell"
	"github.com/gospell/plaintext"
)

// UnknownWord records one unknown word instance
type UnknownWord struct {
	Filename string `desc:"filename only, no path"`                    // gospell.Diff.Filename
	Path     string `desc:"path only, no filename"`                    // gospell.Diff.Path
	Word     string `desc:"unknown word"`                              // gospell.Diff.Original
	Text     string `desc:"full line of text containing unknown word"` // gospell.Diff.Line
	LineNo   int    `desc:"the line number"`                           // gospell.Diff.LineNum
}

func CheckFile(fullpath string) ([]UnknownWord, error) {
	var unknowns []UnknownWord

	path, filename := filepath.Split(fullpath)

	// TODO: comment from gospell authoer -  based on OS (Windows vs. Linux)
	//dictPath := flag.String("path", ".:/usr/local/share/hunspell:/usr/share/hunspell", "Search path for dictionaries")
	dictPath := "/usr/local/share/hunspell"

	// TODO : comment from gospell authoer -  based on environment variable settings
	//dicts := flag.String("d", "en_US", "dictionaries to load")
	dicts := "en_US"

	//personalDict := flag.String("p", "", "personal wordlist file")
	personalDict := ""

	affFile := ""
	dicFile := ""
	for _, base := range filepath.SplitList(dictPath) {
		affFile = filepath.Join(base, dicts+".aff")
		dicFile = filepath.Join(base, dicts+".dic")
		//log.Printf("Trying %s", affFile)
		_, err1 := os.Stat(affFile)
		_, err2 := os.Stat(dicFile)
		if err1 == nil && err2 == nil {
			break
		}
		affFile = ""
		dicFile = ""
	}

	if affFile == "" {
		return unknowns, fmt.Errorf("Unable to load %s. Download dictionaries into /usr/local/share/hunspell", dicts)
	}

	log.Printf("Loading %s %s", affFile, dicFile)
	timeStart := time.Now()
	h, err := gospell.NewGoSpell(affFile, dicFile)
	timeEnd := time.Now()

	// note: 10x too slow
	log.Printf("Loaded in %v", timeEnd.Sub(timeStart))
	if err != nil {
		log.Fatalf("%s", err)
	}

	if personalDict != "" {
		raw, err := ioutil.ReadFile(personalDict)
		if err != nil {
			log.Fatalf("Unable to load personal dictionary %s: %s", personalDict, err)
		}
		duplicates, err := h.AddWordList(bytes.NewReader(raw))
		if err != nil {
			log.Fatalf("Unable to process personal dictionary %s: %s", personalDict, err)
		}
		if len(duplicates) > 0 {
			for _, word := range duplicates {
				log.Printf("Word %q in personal dictionary already exists in main dictionary", word)
			}
		}
	}

	// todo: this will read what is on disk - do we want to pass in the bytes rather than the filename?
	raw, err := ioutil.ReadFile(fullpath)
	if err != nil {
		log.Fatalf("Unable to read Stdin: %s", err)
	}
	pt, _ := plaintext.NewIdentity()
	results := gospell.SpellFile(h, pt, raw)
	for _, diff := range results {
		unknown := UnknownWord{
			Filename: filename,
			Path:     path,
			Text:     diff.Line,
			LineNo:   diff.LineNum,
			Word:     diff.Original,
		}
		unknowns = append(unknowns, unknown)
	}
	return unknowns, nil
}
