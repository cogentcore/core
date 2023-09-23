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
		gi.NewDialog(dialog).Modal().Movable().Closable().Back().Run() // <- winner!

		gi.RunStage(dialog, &gi.StageOpts{Type: gi.BottomSheet, Modal: true})
		// gi.OpenDialog(gi.SceneLib("dialog"))
		// gi.RunStage(&gi.Stage{Scene: dialog, Type: gi.Dialog, Modal: true, Window: true})
	})

	gi.NewWindow(scene).Name("hello").Title("Hello World!").Width(512).Height(384).Run()
}
