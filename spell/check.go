// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spell

import (
	"log"
	"regexp"
	"strings"

	"github.com/goki/pi/lex"
	"github.com/goki/pi/token"
)

var isLetter *regexp.Regexp

func init() {
	var err error
	isLetter, err = regexp.Compile(`^[a-zA-Z\']+$`)
	if err != nil {
		log.Println(err)
	}
}

// IsWord returns true if the string follows rules to accept as word
func IsWord(word string) bool {
	return isLetter.MatchString(word)
}

// CheckLexLine returns the Lex regions for any words that are misspelled
// within given line of text with existing Lex tags -- automatically
// excludes any Code token regions (see token.IsCode).  Token is set
// to token.TextSpellErr on returned Lex's
func CheckLexLine(src []rune, tags lex.Line) lex.Line {
	wrds := tags.NonCodeWords(src)
	var ser lex.Line
	for _, t := range wrds {
		wrd := string(t.Src(src))
		lwrd := isLetter.FindString(wrd)
		if len(lwrd) <= 2 {
			continue
		}
		_, known := CheckWord(lwrd)
		if !known {
			t.Tok.Tok = token.TextSpellErr
			widx := strings.Index(wrd, lwrd)
			t.St += widx
			t.Ed = t.St + len(lwrd) // fix!
			t.Now()
			ser = append(ser, t)
		}
	}
	return ser
}
