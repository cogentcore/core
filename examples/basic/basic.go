package main

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
)

func main() { gimain.Main(mainrun) }

func mainrun() {
	scene := gi.NewScene("hello")
	gi.NewLabel(&scene.Frame, "label").SetText("Hello, World!")

	but := gi.NewButton(&scene.Frame, "open-dialog").SetText("Open Dialog")
	but.OnClicked(func() {
		dialog := gi.NewScene("dialog")
		gi.NewLabel(&dialog.Frame, "dialog").SetText("Dialog!")
		gi.NewDialog(dialog, but).SetModal().SetMovable().SetCloseable().Run()
	})

	gi.NewWindow(scene).
		SetName("hello").
		SetTitle("Hello World!").
		SetWidth(512).SetHeight(384).
		Run().Wait()
}
