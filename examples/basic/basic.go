package main

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
)

func main() { gimain.Main(mainrun) }

func mainrun() {
	scene := gi.NewScene("hello")
	gi.NewLabel(&scene.Frame, "label").SetText("Hello, World!")

	dialog := gi.NewScene("dialog")
	gi.NewLabel(&dialog.Frame, "dialog").SetText("Dialog!")

	but := gi.NewButton(&scene.Frame, "open-dialog").SetText("Open Dialog")
	but.OnClicked(func() {
		// note: but provides context for where to open dialog
		// gi.NewDialog(dialog, but).SetModal().SetMovable().SetClosable().SetBack().Run() // <- winner!
	})

	// note: on Desktop, default is for Window to open in a new RenderWin
	// on Mobile it opens in the one window if the first one.
	// SetOwnWin() explicitly puts separate window

	gi.NewWindow(scene).SetName("hello").SetTitle("Hello World!").SetWidth(512).SetHeight(384).Run()

	// todo: could provide a wrapper
	gi.WinWait.Wait() // wait for windows to close
}
