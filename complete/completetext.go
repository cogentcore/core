package complete

import (
	"github.com/goki/gi/spell"
)

// Complete Tex is the function for completing .tex files
func CompleteText(s string) (result []string, err error) {
	result, err = spell.Complete(s)
	return result, err
}

// EditText is a chance to modify the completion selection before it is inserted
func EditText(text string, cp int, completion string, seed string) (newText string, delta int) {
	return completion, 0
}
