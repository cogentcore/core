// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package texteditor

import (
	"fmt"

	"cogentcore.org/core/core"
	"cogentcore.org/core/parse"
	"cogentcore.org/core/parse/complete"
	"cogentcore.org/core/parse/lexer"
	"cogentcore.org/core/parse/parser"
	"cogentcore.org/core/spell"
	"cogentcore.org/core/texteditor/textbuf"
)

// CompleteParse uses [parse] symbols and language; the string is a line of text
// up to point where user has typed.
// The data must be the *FileState from which the language type is obtained.
func CompleteParse(data any, text string, posLine, posChar int) (md complete.Matches) {
	sfs := data.(*parse.FileStates)
	if sfs == nil {
		// log.Printf("CompletePi: data is nil not FileStates or is nil - can't complete\n")
		return md
	}
	lp, err := parse.LangSupport.Properties(sfs.Sup)
	if err != nil {
		// log.Printf("CompletePi: %v\n", err)
		return md
	}
	if lp.Lang == nil {
		return md
	}

	// note: must have this set to ture to allow viewing of AST
	// must set it in pi/parse directly -- so it is changed in the fileparse too
	parser.GUIActive = true // note: this is key for debugging -- runs slower but makes the tree unique

	md = lp.Lang.CompleteLine(sfs, text, lexer.Pos{posLine, posChar})
	return md
}

// CompleteEditParse uses the selected completion to edit the text.
func CompleteEditParse(data any, text string, cursorPos int, comp complete.Completion, seed string) (ed complete.Edit) {
	sfs := data.(*parse.FileStates)
	if sfs == nil {
		// log.Printf("CompleteEditPi: data is nil not FileStates or is nil - can't complete\n")
		return ed
	}
	lp, err := parse.LangSupport.Properties(sfs.Sup)
	if err != nil {
		// log.Printf("CompleteEditPi: %v\n", err)
		return ed
	}
	if lp.Lang == nil {
		return ed
	}
	return lp.Lang.CompleteEdit(sfs, text, cursorPos, comp, seed)
}

// LookupParse uses [parse] symbols and language; the string is a line of text
// up to point where user has typed.
// The data must be the *FileState from which the language type is obtained.
func LookupParse(data any, text string, posLine, posChar int) (ld complete.Lookup) {
	sfs := data.(*parse.FileStates)
	if sfs == nil {
		// log.Printf("LookupPi: data is nil not FileStates or is nil - can't lookup\n")
		return ld
	}
	lp, err := parse.LangSupport.Properties(sfs.Sup)
	if err != nil {
		// log.Printf("LookupPi: %v\n", err)
		return ld
	}
	if lp.Lang == nil {
		return ld
	}

	// note: must have this set to ture to allow viewing of AST
	// must set it in pi/parse directly -- so it is changed in the fileparse too
	parser.GUIActive = true // note: this is key for debugging -- runs slower but makes the tree unique

	ld = lp.Lang.Lookup(sfs, text, lexer.Pos{posLine, posChar})
	if len(ld.Text) > 0 {
		TextDialog(nil, "Lookup: "+text, string(ld.Text))
		return ld
	}
	if ld.Filename != "" {
		txt := textbuf.FileRegionBytes(ld.Filename, ld.StLine, ld.EdLine, true, 10) // comments, 10 lines back max
		prmpt := fmt.Sprintf("%v [%d:%d]", ld.Filename, ld.StLine, ld.EdLine)
		TextDialog(nil, "Lookup: "+text+": "+prmpt, string(txt))
		return ld
	}

	return ld
}

// CompleteText does completion for text files.
func CompleteText(data any, text string, posLine, posChar int) (md complete.Matches) {
	err := InitSpell() // text completion uses the spell code to generate completions and suggestions
	if err != nil {
		fmt.Printf("Could not initialize spelling model: Spelling model needed for text completion: %v", err)
		return md
	}

	md.Seed = complete.SeedSpace(text)
	if md.Seed == "" {
		return md
	}
	result := spell.Complete(md.Seed)
	possibles := complete.MatchSeedString(result, md.Seed)
	for _, p := range possibles {
		m := complete.Completion{Text: p}
		md.Matches = append(md.Matches, m)
	}
	return md
}

// CompleteTextEdit uses the selected completion to edit the text
func CompleteTextEdit(data any, text string, cursorPos int, completion complete.Completion, seed string) (ed complete.Edit) {
	ed = core.CompleteEditText(text, cursorPos, completion.Text, seed)
	return ed
}
