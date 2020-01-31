package giv

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/goki/gi/gi"
	"github.com/goki/ki/ints"
	"github.com/goki/pi/complete"
	"github.com/goki/pi/lex"
	"github.com/goki/pi/parse"
	"github.com/goki/pi/pi"
)

// CompletePi uses GoPi symbols and language -- the string is a line of text
// up to point where user has typed.
// The data must be the *FileState from which the language type is obtained.
func CompletePi(data interface{}, text string, posLn, posCh int) (md complete.Matches) {
	sfs := data.(*pi.FileStates)
	if sfs == nil {
		log.Printf("CompletePi: data is nil not FileStates or is nil - can't complete\n")
		return md
	}
	lp, err := pi.LangSupport.Props(sfs.Sup)
	if err != nil {
		log.Printf("CompletePi: %v\n", err)
		return md
	}
	if lp.Lang == nil {
		return md
	}

	// note: must have this set to ture to allow viewing of AST
	// must set it in pi/parse directly -- so it is changed in the fileparse too
	parse.GuiActive = true // note: this is key for debugging -- runs slower but makes the tree unique

	md = lp.Lang.CompleteLine(sfs, text, lex.Pos{posLn, posCh})
	return md
}

// CompleteEditPi uses the selected completion to edit the text
func CompleteEditPi(data interface{}, text string, cursorPos int, comp complete.Completion, seed string) (ed complete.Edit) {
	sfs := data.(*pi.FileStates)
	if sfs == nil {
		log.Printf("CompleteEditPi: data is nil not FileStates or is nil - can't complete\n")
		return ed
	}
	lp, err := pi.LangSupport.Props(sfs.Sup)
	if err != nil {
		log.Printf("CompleteEditPi: %v\n", err)
		return ed
	}
	if lp.Lang == nil {
		return ed
	}
	return lp.Lang.CompleteEdit(sfs, text, cursorPos, comp, seed)
}

// LookupPi uses GoPi symbols and language -- the string is a line of text
// up to point where user has typed.
// The data must be the *FileState from which the language type is obtained.
func LookupPi(data interface{}, text string, posLn, posCh int) (ld complete.Lookup) {
	sfs := data.(*pi.FileStates)
	if sfs == nil {
		log.Printf("LookupPi: data is nil not FileStates or is nil - can't complete\n")
		return ld
	}
	lp, err := pi.LangSupport.Props(sfs.Sup)
	if err != nil {
		log.Printf("LookupPi: %v\n", err)
		return ld
	}
	if lp.Lang == nil {
		return ld
	}

	// note: must have this set to ture to allow viewing of AST
	// must set it in pi/parse directly -- so it is changed in the fileparse too
	parse.GuiActive = true // note: this is key for debugging -- runs slower but makes the tree unique

	ld = lp.Lang.Lookup(sfs, text, lex.Pos{posLn, posCh})
	if len(ld.Text) > 0 {
		TextViewDialog(nil, ld.Text, DlgOpts{Title: "Lookup: " + text})
		return ld
	}
	if ld.Filename != "" {
		fp, err := os.Open(ld.Filename)
		if err != nil {
			log.Println(err)
			return ld
		}
		txt, err := ioutil.ReadAll(fp)
		fp.Close()
		if err != nil {
			log.Println(err)
			return ld
		}
		if ld.StLine > 0 || ld.EdLine > 0 {
			lns := bytes.Split(txt, []byte("\n"))
			nln := len(lns)
			if ld.EdLine > 0 && ld.EdLine > ld.StLine && ld.EdLine < nln {
				el := ints.MinInt(ld.EdLine+1, nln-1)
				lns = lns[:el]
			}
			if ld.StLine > 0 && ld.StLine < len(lns) {
				lns = lns[ld.StLine:]
			}
			txt = bytes.Join(lns, []byte("\n"))
			txt = append(txt, '\n')
		}
		prmpt := fmt.Sprintf("%v [%d:%d]", ld.Filename, ld.StLine, ld.EdLine)
		TextViewDialog(nil, []byte(txt), DlgOpts{Title: "Lookup: " + text, Prompt: prmpt, Filename: ld.Filename, LineNos: true})
		return ld
	}

	return ld
}

// CompleteText does completion for text files
func CompleteText(data interface{}, text string, posLn, posCh int) (md complete.Matches) {
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
func CompleteTextEdit(data interface{}, text string, cursorPos int, completion complete.Completion, seed string) (ed complete.Edit) {
	ed = gi.CompleteEditText(text, cursorPos, completion.Text, seed)
	return ed
}
