package complete

import (
	"github.com/goki/gi/spell"
)

// CompleteText is the function for completing text files
func CompleteText(s string) (result []string, err error) {
	result, err = spell.Complete(s)
	return result, err
}

// EditText is a chance to modify the completion selection before it is inserted
func EditText(text string, cp int, completion string, seed string) (ed EditData) {
	ed.NewText = completion
	return ed
}
