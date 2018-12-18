package giv

import (
	"github.com/goki/gi/spell"
)

// SpellCorrectEdit uses the selected correction to edit the text
func SpellCorrectEdit(data interface{}, nwstr string, old string) (ed spell.EditData) {
	ed = spell.CorrectText(old, nwstr)
	return ed
}
