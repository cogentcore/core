package main

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
)

func main() { gimain.Run(app) }

func app() {
	// gi.WinEventTrace = true
	// gi.EventTrace = true
	gi.LayoutTrace = true
	gi.LayoutTraceDetail = true
	// gi.RenderTrace = true

	sc := gi.NewScene().SetTitle("Basic")

	gi.DefaultTopAppBar = nil

	// gi.NewIcon(sc).SetIcon(icons.Add)

	// gi.NewLabel(sc).SetText("Hello, World!")

	// gi.NewButton(sc).
	// 	SetText("Open Dialog").SetIcon(icons.OpenInNew).
	// 	OnClick(func(e events.Event) {
	// 		fmt.Println("button clicked")
	// 		// dialog := gi.NewScene("dialog")
	// 		// gi.NewLabel(&dialog.Frame, "dialog").SetText("Dialog!")
	// 		// gi.NewDialog(dialog, but).SetModal().SetMovable().SetCloseable().Run()
	// 	})

	// gi.NewSwitch(sc)

	gi.NewWindow(sc).Run().Wait()
}
