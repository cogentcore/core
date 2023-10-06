package main

import (
	"fmt"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/goosi/events"
	"goki.dev/icons"
)

func main() { gimain.Run(app) }

func app() {
	gi.WinEventTrace = true
	gi.EventTrace = true
	gi.LayoutTrace = true
	gi.RenderTrace = true

	scene := gi.NewScene("hello").SetTitle("Hello World!")
	gi.NewLabel(&scene.Frame).SetText("Hello, World!")

	gi.NewButton(&scene.Frame).
		SetText("Open Dialog").SetIcon(icons.OpenInNew).
		On(events.Click, func(e events.Event) {
			fmt.Println("button clicked")
			// dialog := gi.NewScene("dialog")
			// gi.NewLabel(&dialog.Frame, "dialog").SetText("Dialog!")
			// gi.NewDialog(dialog, but).SetModal().SetMovable().SetCloseable().Run()
		})

	gi.NewWindow(scene).Run().Wait()
}
