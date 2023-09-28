package main

import (
	"fmt"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/icons"
)

func main() { gimain.Main(mainrun) }

func mainrun() {
	// gi.WinEventTrace = true
	// gi.EventTrace = true
	// gi.LayoutTrace = true
	// gi.RenderTrace = true

	scene := gi.NewScene("hello")
	gi.NewLabel(&scene.Frame, "label").SetText("Hello, World!")

	but := gi.NewButton(&scene.Frame, "open-dialog").
		SetText("Open Dialog").
		SetIcon(icons.OpenInNew)

	but.OnClicked(func() {
		fmt.Println("button clicked")
		// dialog := gi.NewScene("dialog")
		// gi.NewLabel(&dialog.Frame, "dialog").SetText("Dialog!")
		// gi.NewDialog(dialog, but).SetModal().SetMovable().SetCloseable().Run()
	})

	gi.NewWindow(scene).
		SetName("hello").
		SetTitle("Hello World!").
		SetWidth(512).SetHeight(384).
		Run().Wait()
}
