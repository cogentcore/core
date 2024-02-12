package main

import (
	"cogentcore.org/core/gi"
	_ "cogentcore.org/core/giv"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/styles"
)

func main() {
	b := gi.NewBody("Hello")
	gi.NewButton(b).SetText("Hello, World!")

	gi.NewTextField(b).SetText("One line textfield with a relatively long initial text")
	gi.NewTextField(b).SetLeadingIcon(icons.Search)
	gi.NewTextField(b).AddClearButton()
	gi.NewTextField(b).SetText("Multiline textfield with a relatively long initial text").
		Style(func(s *styles.Style) {
			s.SetTextWrap(true)
			s.Max.X.Em(10)
		})
	b.RunMainWindow()
}
