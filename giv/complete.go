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
func CompleteGo(data interface{}, text string, pos token.Position) (matches complete.Completions, seed string) {
	var txbuf *TextBuf
	switch t := data.(type) {
	case *TextView:
		txbuf = t.Buf
	}
	if txbuf == nil {
		log.Printf("complete.Complete: txbuf is nil - can't complete\n")
		return
	}

	seed = complete.SeedGolang(text)
	textbytes := make([]byte, 0, txbuf.NLines*40)
	for _, lr := range txbuf.Lines {
		textbytes = append(textbytes, []byte(string(lr))...)
		textbytes = append(textbytes, '\n')
	}
	results := complete.CompleteGo(textbytes, pos)
	return results, seed
}

// CompleteGoEdit uses the selected completion to edit the text
func CompleteGoEdit(data interface{}, text string, cursorPos int, selection string, seed string) (s string, delta int) {
	s, delta = complete.EditGoCode(text, cursorPos, selection, seed)
	return s, delta
}

// CompleteText does completion for text files
func CompleteText(data interface{}, text string, pos token.Position) (matches complete.Completions, seed string) {
	err := gi.InitSpell() // text completion uses the spell code to generate completions and suggestions
	if err != nil {
		fmt.Println("Could not initialize spelling model: Spelling model needed for text completion: %v", err)
		return nil, seed
	}

	seed = complete.SeedWhiteSpace(text)
	if seed == "" {
		return nil, seed
	}
	result, err := complete.CompleteText(seed)
	if err != nil {
		fmt.Println("Error completing text: %v", err)
		return nil, seed
	}
	possibles := complete.MatchSeedString(result, seed)
	for _, p := range possibles {
		m := complete.Completion{Text: p, Icon: ""}
		matches = append(matches, m)
	}
	return matches, seed
}

// CompleteTextEdit uses the selected completion to edit the text
func CompleteTextEdit(data interface{}, text string, cursorPos int, selection string, seed string) (s string, delta int) {
	s, delta = complete.EditText(text, cursorPos, selection, seed)
	return s, delta
}
