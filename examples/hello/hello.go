package main

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
)

func main() { gimain.Main(mainrun) }

func mainrun() {
	scene := gi.NewScene("hello")
	gi.NewLabel(&scene.Frame).SetText("Hello, World!")
	gi.NewWindow(scene).SetName("hello").Run().Wait()
}
