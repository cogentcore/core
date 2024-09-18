// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package transpile

import (
	"fmt"
	"strings"
	"unicode"
)

func ExecWords(ln string) ([]string, error) {
	ln = strings.TrimSpace(ln)
	n := len(ln)
	if n == 0 {
		return nil, nil
	}

	if ln[0] == '$' {
		ln = strings.TrimSpace(ln[1:])
		n = len(ln)
		if n == 0 {
			return nil, nil
		}
		if ln[n-1] == '$' {
			ln = strings.TrimSpace(ln[:n-1])
			n = len(ln)
			if n == 0 {
				return nil, nil
			}
		}
	}

	word := ""
	esc := false
	dQuote := false
	bQuote := false
	brace := 0
	brack := 0
	redir := false

	var words []string
	addWord := func() {
		if brace > 0 { // always accum into one token inside brace
			return
		}
		if len(word) > 0 {
			words = append(words, word)
			word = ""
		}
	}

	atStart := true
	sbrack := (ln[0] == '[')
	if sbrack {
		word = "["
		addWord()
		brack++
		ln = ln[1:]
		atStart = false
	}

	for _, r := range ln {
		quote := dQuote || bQuote

		if redir {
			redir = false
			if r == '&' {
				word += string(r)
				addWord()
				continue
			}
			if r == '>' {
				word += string(r)
				redir = true
				continue
			}
			addWord()
		}

		switch {
		case esc:
			if brace == 0 && unicode.IsSpace(r) { // we will be quoted later anyway
				word = word[:len(word)-1]
			}
			word += string(r)
			esc = false
		case r == '\\':
			esc = true
			word += string(r)
		case r == '"':
			if !bQuote {
				dQuote = !dQuote
			}
			word += string(r)
		case r == '`':
			if !dQuote {
				bQuote = !bQuote
			}
			word += string(r)
		case quote: // absorbs quote -- no need to check below
			word += string(r)
		case unicode.IsSpace(r):
			addWord()
			continue // don't reset at start
		case r == '{':
			if brace == 0 {
				addWord()
				word = "{"
				addWord()
			}
			brace++
		case r == '}':
			brace--
			if brace == 0 {
				addWord()
				word = "}"
				addWord()
			}
		case r == '[':
			word += string(r)
			if atStart && brack == 0 {
				sbrack = true
				addWord()
			}
			brack++
		case r == ']':
			brack--
			if brack == 0 && sbrack { // only point of tracking brack is to get this end guy
				addWord()
				word = "]"
				addWord()
			} else {
				word += string(r)
			}
		case r == '<' || r == '>' || r == '|':
			addWord()
			word += string(r)
			redir = true
		case r == '&': // known to not be redir
			addWord()
			word += string(r)
		case r == ';':
			addWord()
			word += string(r)
			addWord()
			atStart = true
			continue // avoid reset
		default:
			word += string(r)
		}
		atStart = false
	}
	addWord()
	if dQuote || bQuote || brack > 0 {
		return words, fmt.Errorf("goal: exec command has unterminated quotes (\": %v, `: %v) or brackets [ %v ]", dQuote, bQuote, brack > 0)
	}
	return words, nil
}

// ExecWordIsCommand returns true if given exec word is a command-like string
// (excluding any paths)
func ExecWordIsCommand(f string) bool {
	if strings.Contains(f, "(") || strings.Contains(f, "=") {
		return false
	}
	return true
}
