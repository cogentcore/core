package main

import "cogentcore.org/core/gi"

func main() {
	b := gi.NewAppBody("Hello")
	gi.NewButton(b).SetText("Hello, World!")
	b.RunMainWindow()
}
