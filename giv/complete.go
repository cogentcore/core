package giv

import (
	"fmt"
	"go/token"
	"log"

	"github.com/goki/gi/gi"
	"github.com/goki/pi/complete"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/pi"
)

// CompletePi uses GoPi symbols and language -- the string is a line of text
// up to point where user has typed.
// The data must be the *FileState from which the language type is obtained.
func CompletePi(data interface{}, text string, pos token.Position) (md complete.MatchData) {
	sfs := data.(*pi.FileState)
	if sfs == nil {
		log.Printf("CompletePi: data is nil not FileState or is nil - can't complete\n")
		return md
	}
	lp, err := pi.LangSupport.Props(sfs.Src.Sup)
	if err != nil {
		log.Printf("CompletePi: %v\n", err)
		return md
	}
	if lp.Lang == nil {
		return md
	}
	return lp.Lang.CompleteLine(sfs, text, lex.Pos{pos.Line, pos.Column})
}

// CompleteEditPi uses the selected completion to edit the text
func CompleteEditPi(data interface{}, text string, cursorPos int, comp complete.Completion, seed string) (ed complete.EditData) {
	sfs := data.(*pi.FileState)
	if sfs == nil {
		log.Printf("CompleteEditPi: data is nil not FileState or is nil - can't complete\n")
		return ed
	}
	lp, err := pi.LangSupport.Props(sfs.Src.Sup)
	if err != nil {
		log.Printf("CompleteEditPi: %v\n", err)
		return ed
	}
	if lp.Lang == nil {
		return ed
	}
	return lp.Lang.CompleteEdit(sfs, text, cursorPos, comp, seed)
}

// CompleteText does completion for text files
func CompleteText(data interface{}, text string, pos token.Position) (md complete.MatchData) {
	err := gi.InitSpell() // text completion uses the spell code to generate completions and suggestions
	if err != nil {
		fmt.Printf("Could not initialize spelling model: Spelling model needed for text completion: %v", err)
		return md
	}

	md.Seed = complete.SeedWhiteSpace(text)
	if md.Seed == "" {
		return md
	}
	result, err := gi.CompleteText(md.Seed)
	if err != nil {
		fmt.Printf("Error completing text: %v", err)
		return md
	}
	possibles := complete.MatchSeedString(result, md.Seed)
	for _, p := range possibles {
		m := complete.Completion{Text: p, Icon: ""}
		md.Matches = append(md.Matches, m)
	}
	return md
}

// CompleteTextEdit uses the selected completion to edit the text
func CompleteTextEdit(data interface{}, text string, cursorPos int, completion complete.Completion, seed string) (ed complete.EditData) {
	ed = gi.CompleteEditText(text, cursorPos, completion.Text, seed)
	return ed
}
