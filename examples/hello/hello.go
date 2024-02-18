package main

import "cogentcore.org/core/gi"

func main() {
	b := gi.NewBody("Hello")
	gi.NewButton(b).SetText("Hello, World!")
	b.RunMainWindow()
}
