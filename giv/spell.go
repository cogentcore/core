package giv

import (
	"github.com/goki/gi/spell"
)

// SpellCorrectEdit uses the selected correction to edit the text
func SpellCorrectEdit(data interface{}, text string, cursorPos int, new string, old string) (s string, delta int) {
	s, delta = spell.CorrectText(text, cursorPos, old, new)
	return s, delta
}
