package complete

import (
	"fmt"
	"go/token"
	"strings"
	"unicode"
)

// Complete Tex is the function for completing .tex files
func CompleteTex(bytes []byte, pos token.Position) []Completion {
	var results []Completion

	fmt.Println("Completion for tex files not implemented")
	return results
}

// EditTex replaces the completion seed and any text up to the next whitespace or other go delimiter
// with the selected completion. delta is the change in cursor position (cp).
func EditTex(text string, cp int, completion string, seed string) (newText string, delta int) {
	s1 := string(text[0:cp])
	s2 := string(text[cp:])

	if len(s2) > 0 {
		r := rune(s2[0])
		// find the next whitespace or end of text
		if !(unicode.IsSpace(r)) {
			count := len(s2)
			for i, c := range s2 {
				r = rune(c)
				if unicode.IsSpace(r) || r == rune('(') || r == rune('.') || r == rune('[') {
					s2 = s2[i:]
					break
				}
				// might be last word
				if i == count-1 {
					s2 = ""
				}
			}
		}
	}

	s1 = strings.TrimSuffix(s1, seed)
	s1 += completion
	t := s1 + s2
	delta = len(completion) - len(seed)
	return t, delta
}
