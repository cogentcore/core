package main

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/gi/v2/giv"
	"goki.dev/mat32/v2"
)

func main() { gimain.Run(app) }

type Wide struct {
	Name  string
	Title string
	F2    string
	F3    string
}

type Test struct {
	Wide Wide `view:"inline"`
	Vec  mat32.Vec2
}

func app() {
	// gi.WinEventTrace = true
	// gi.EventTrace = true
	gi.LayoutTrace = true
	gi.LayoutTraceDetail = true
	// gi.RenderTrace = true

	sc := gi.NewScene().SetTitle("Basic")

	gi.DefaultTopAppBar = nil

	ts := &Test{}
	giv.NewStructView(sc).SetStruct(ts)

	// gi.NewTextField(sc).AddClearButton()

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
