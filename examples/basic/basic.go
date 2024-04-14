package main

import "cogentcore.org/core/core"

func main() {
	b := core.NewBody("Hello")
	core.NewButton(b).SetText("Hello, World!")
	b.RunMainWindow()
}
