package main

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
)

func main() { gimain.Run(app) }

func app() {
	// gi.WinEventTrace = true
	// gi.EventTrace = true
	// gi.LayoutTrace = true
	// gi.RenderTrace = true

	sc := gi.NewScene().SetTitle("Basic")
	sc.TopAppBar = func(*gi.Toolbar) {}
	// gi.NewLabel(scene).SetText("Hello, World!")

	// gi.NewButton(scene).
	// 	SetText("Open Dialog").SetIcon(icons.OpenInNew).
	// 	OnClick(func(e events.Event) {
	// 		fmt.Println("button clicked")
	// 		// dialog := gi.NewScene("dialog")
	// 		// gi.NewLabel(&dialog.Frame, "dialog").SetText("Dialog!")
	// 		// gi.NewDialog(dialog, but).SetModal().SetMovable().SetCloseable().Run()
	// 	})

	gi.NewSwitch(sc)

	gi.NewWindow(sc).Run().Wait()
}
