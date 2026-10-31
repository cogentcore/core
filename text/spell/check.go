// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spell

import (
	"strings"

	"cogentcore.org/core/text/parse/lexer"
	"cogentcore.org/core/text/token"
)

// CheckLexLine returns the Lex regions for any words that are misspelled
// within given line of text with existing Lex tags -- automatically
// excludes any Code token regions (see token.IsCode).  Token is set
// to token.TextSpellErr on returned Lex's
func CheckLexLine(src []rune, tags lexer.Line) lexer.Line {
	wrds := tags.NonCodeWords(src)
	var lexs lexer.Line
	var words []string
	for _, t := range wrds {
		wrd := string(t.Src(src))
		lwrd := lexer.FirstWordApostrophe(wrd)
		if len(lwrd) <= 2 {
			continue
		}
		widx := strings.Index(wrd, lwrd)
		ld := len(wrd) - len(lwrd)
		t.Start += widx
		t.End += widx - ld
		lexs = append(lexs, t)
		words = append(words, lwrd)
	}
	var ser lexer.Line
	for _, i := range Spell.Misspelled(words) {
		t := lexs[i]
		t.Token.Token = token.TextSpellErr
		t.Now()
		ser = append(ser, t)
	}
	return ser
}
