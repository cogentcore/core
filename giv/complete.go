package giv

import (
	"fmt"
	"go/token"
	"log"

	"github.com/goki/gi"
	"github.com/goki/gi/complete"
)

// CompleteGo uses github.com/mdempsky/gocode to do code completion
// gocode requires the entire file and the position of the cursor within the file
func CompleteGo(data interface{}, text string, pos token.Position) (md complete.MatchData) {
	var txbuf *TextBuf
	switch t := data.(type) {
	case *TextView:
		txbuf = t.Buf
	case *TextBuf:
		txbuf = t
	}
	if txbuf == nil {
		log.Printf("complete.Complete: txbuf is nil - can't complete\n")
		return md
	}

	md.Seed = complete.SeedGolang(text)
	textbytes := make([]byte, 0, txbuf.NLines*40)
	for _, lr := range txbuf.Lines {
		textbytes = append(textbytes, []byte(string(lr))...)
		textbytes = append(textbytes, '\n')
	}
	md.Matches = complete.CompleteGo(textbytes, pos)
	return md
}

// CompleteGoEdit uses the selected completion to edit the text
func CompleteGoEdit(data interface{}, text string, cursorPos int, completion complete.Completion, seed string) (ed complete.EditData) {
	ed = complete.EditGoCode(text, cursorPos, completion, seed)
	return ed
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
	result, err := complete.CompleteText(md.Seed)
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
	ed = complete.EditText(text, cursorPos, completion.Text, seed)
	return ed
}
