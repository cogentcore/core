// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spell

import (
	"strings"

	"cogentcore.org/core/pi/lexer"
	"cogentcore.org/core/pi/token"
)

// CheckLexLine returns the Lex regions for any words that are misspelled
// within given line of text with existing Lex tags -- automatically
// excludes any Code token regions (see token.IsCode).  Token is set
// to token.TextSpellErr on returned Lex's
func CheckLexLine(src []rune, tags lexer.Line) lexer.Line {
	wrds := tags.NonCodeWords(src)
	var ser lexer.Line
	for _, t := range wrds {
		wrd := string(t.Src(src))
		lwrd := lexer.FirstWordApostrophe(wrd)
		if len(lwrd) <= 2 {
			continue
		}
		_, known := CheckWord(lwrd)
		if !known {
			t.Token.Token = token.TextSpellErr
			widx := strings.Index(wrd, lwrd)
			ld := len(wrd) - len(lwrd)
			t.St += widx
			t.Ed += widx - ld
			t.Now()
			ser = append(ser, t)
		}
	}
	return ser
}
