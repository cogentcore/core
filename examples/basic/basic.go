package main

import (
	"github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
)

func main() { gimain.Main(mainrun) }

func mainrun() {
	scene := gi.NewScene("hello")
	label := gi.NewLabel(scene, "label")
	label.Text = "Hello, World!"

	dialog := gi.NewScene("dialog")
	label = gi.NewLabel(dialog, "dialog")
	label.Text = "Dialog!"

	but := gi.NewButton(scene, "open-dialog")
	but.Text = "Open Dialog"
	but.OnClicked(func() {
		gi.RunStage(dialog, &gi.StageOpts{Type: gi.Dialog, Modal: true, Movable: true, Closeable: true, Back: true})
		gi.RunStage(dialog, gi.StageType(gi.Dialog), gi.StageModal(), gi.StageMovable(), gi.StageClosable(), gi.StageBack())
		gi.NewDialog(dialog).SetModal().SetMovable().SetClosable().SetBack().Run() // <- winner!

		gi.RunStage(dialog, &gi.StageOpts{Type: gi.BottomSheet, Modal: true})
		// gi.OpenDialog(gi.SceneLib("dialog"))
		// gi.RunStage(&gi.Stage{Scene: dialog, Type: gi.Dialog, Modal: true, OSWin: true})
	})

	// note: on Desktop, default is for Window to open in a new OSWin
	// on Mobile it opens in the one window if the first one.
	// SetOwnWin() explicitly puts separate window

	win := gi.NewWindow(scene).SetName("hello").SetTitle("Hello World!").SetWidth(512).SetHeight(384).Run()
}
