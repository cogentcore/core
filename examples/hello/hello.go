package main

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
)

func main() { gimain.Run(app) }

func app() {
	scene := gi.NewScene()
	gi.NewLabel(scene).SetText("Hello, World!")
	gi.NewWindow(scene).Run().Wait()
}
