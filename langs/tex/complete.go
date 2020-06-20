// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tex

import (
	"strings"
	"unicode"

	"github.com/goki/pi/complete"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/pi"
)

func (tl *TexLang) CompleteLine(fss *pi.FileStates, str string, pos lex.Pos) (md complete.Matches) {
	origStr := str
	lfld := lex.LastField(str)
	str = lex.LastScopedString(str)
	if lfld[0] == '\\' && lfld[1:] == str { // use the /
		str = lfld
	}
	if HasCite(lfld) {
		return tl.CompleteCite(fss, origStr, str, pos)
	}
	md.Seed = str
	if len(LaTeXCmdsAll) == 0 {
		LaTeXCmdsAll = append(LaTeXCmds, CiteCmds...)
	}
	for _, ls := range LaTeXCmdsAll {
		if strings.HasPrefix(ls, str) {
			c := complete.Completion{Text: ls, Label: ls, Icon: "function"}
			md.Matches = append(md.Matches, c)
		}
	}
	return md
}

// Lookup is the main api called by completion code in giv/complete.go to lookup item
func (tl *TexLang) Lookup(fss *pi.FileStates, str string, pos lex.Pos) (ld complete.Lookup) {
	origStr := str
	lfld := lex.LastField(str)
	str = lex.LastScopedString(str)
	if HasCite(lfld) {
		return tl.LookupCite(fss, origStr, str, pos)
	}
	return
}

func (tl *TexLang) CompleteEdit(fss *pi.FileStates, text string, cp int, comp complete.Completion, seed string) (ed complete.Edit) {
	// if the original is ChildByName() and the cursor is between d and B and the comp is Children,
	// then delete the portion after "Child" and return the new comp and the number or runes past
	// the cursor to delete
	s2 := text[cp:]
	// gotParen := false
	if len(s2) > 0 && lex.IsLetterOrDigit(rune(s2[0])) {
		for i, c := range s2 {
			if c == '{' {
				// gotParen = true
				s2 = s2[:i]
				break
			}
			isalnum := c == '_' || unicode.IsLetter(c) || unicode.IsDigit(c)
			if !isalnum {
				s2 = s2[:i]
				break
			}
		}
	} else {
		s2 = ""
	}

	var nw = comp.Text
	// if gotParen && strings.HasSuffix(nw, "()") {
	// 	nw = nw[:len(nw)-2]
	// }

	// fmt.Printf("text: %v|%v  comp: %v  s2: %v\n", text[:cp], text[cp:], nw, s2)
	ed.NewText = nw
	ed.ForwardDelete = len(s2)
	return ed
}

// CiteCmds is a list of latex citation commands (APA style requires many variations).
// We include all the variations so they show up in completion.
var CiteCmds = []string{`\cite`, `\citep`, `\citet`, `\citeNP`, `\citeyearpar`, `\citeyear`, `\citeauthor`, `\citeA`, `\citealp`, `\citeyearNP`, `\parencite`, `\textcite`, `\nptextcite`, `\incite`, `\nopcite`, `\yrcite`, `\yrnopcite`, `\abbrevcite`, `\abbrevincite`}

// HasCite returns true if string has Prefix in CiteCmds
func HasCite(str string) bool {
	for _, cc := range CiteCmds {
		if strings.HasPrefix(str, cc) {
			return true
		}
	}
	return false
}

// LaTeXCmdsAll concatenates LaTeXCmds and CiteCmds
var LaTeXCmdsAll []string

// LaTeXCmds is a big list of standard commands
var LaTeXCmds = []string{
	`\em`,
	`\emph`,
	`\textbf`,
	`\textit`,
	`\texttt`,
	`\textsf`,
	`\textrm`,
	`\tiny`,
	`\scriptsize`,
	`\footnotesize`,
	`\small`,
	`\normalsize`,
	`\large`,
	`\Large`,
	`\LARGE`,
	`\huge`,
	`\Huge`,
	`\begin`,
	`\end`,
	`enumerate`,
	`itemize`,
	`description`,
	`\item`,
	`figure`,
	`table`,
	`tabular`,
	`array`,
	`\hline`,
	`\cline`,
	`\multicolumn`,
	`equation`,
	`center`,
	`\centering`,
	`\verb`,
	`verbatim`,
	`quote`,
	`\section`,
	`\subsection`,
	`\subsubsection`,
	`\paragraph`,
}
