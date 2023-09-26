package main

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
)

func main() { gimain.Main(mainrun) }

func mainrun() {
	scene := gi.NewScene("hello")
	label := gi.NewLabel(scene, "label").SetText("Hello, World!")

	dialog := gi.NewScene("dialog")
	label = gi.NewLabel(dialog, "dialog").SetText("Dialog!")

	but := gi.NewButton(scene, "open-dialog").SetText("Open Dialog")
	but.OnClicked(func() {
		gi.NewDialog(dialog).SetModal().SetMovable().SetClosable().SetBack().Run() // <- winner!
	})

	// note: on Desktop, default is for Window to open in a new RenderWin
	// on Mobile it opens in the one window if the first one.
	// SetOwnWin() explicitly puts separate window

	win := gi.NewWindow(scene).SetName("hello").SetTitle("Hello World!").SetWidth(512).SetHeight(384).Run()

	// todo: could provide a wrapper
	gi.WinWait.Wait() // wait for windows to close
}
