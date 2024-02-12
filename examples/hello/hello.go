package main

import (
	"cogentcore.org/core/gi"
	"cogentcore.org/core/icons"
)

func main() {
	b := gi.NewBody("Hello")
	// gi.NewButton(b).SetText("Hello, World!")
	gi.NewTextField(b).AddClearButton().SetLeadingIcon(icons.Search)
	b.RunMainWindow()
}
