package main

import "cogentcore.org/core/gi"

func main() {
	b := gi.NewAppBody("Hello")
	gi.NewLabel(b).SetText("Hello, World!")
	b.StartMainWindow()
}
