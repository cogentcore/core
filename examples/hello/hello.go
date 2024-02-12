package main

import (
	"cogentcore.org/core/gi"
	_ "cogentcore.org/core/giv"
)

func main() {
	b := gi.NewBody("Hello")
	// gi.NewButton(b).SetText("Hello, World!")

	// gi.NewTextField(b).SetText("One line textfield with a relatively long initial text")
	// gi.NewTextField(b).SetLeadingIcon(icons.Search)
	// gi.NewTextField(b).AddClearButton() // .SetText("This is pretty good")
	gi.NewTextField(b).SetText("Multiline textfield with a relatively long initial text").
		Style(func(s *styles.Style) {
			s.SetTextWrap(true)
			s.Max.X.Em(10)
		})
	b.RunMainWindow()
}
