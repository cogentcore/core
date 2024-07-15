package main

import "cogentcore.org/core/core"

func main() {
	b := core.NewBody()
	core.NewButton(b).SetText("Hello, World!")
	b.RunMainWindow()
}
