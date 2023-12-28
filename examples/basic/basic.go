package main

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
	"goki.dev/goosi/events"
	"goki.dev/icons"
)

func main() { gimain.Run(app) }

func app() {
	b := gi.NewAppBody("basic")
	bt := gi.NewButton(b).SetText("Open dialog").SetIcon(icons.OpenInNew)
	bt.OnClick(func(e events.Event) {
		gi.NewBody().AddTitle("Hello, world!").AddOkOnly().NewDialog(bt).Run()
	})
	b.NewWindow().Run().Wait()
}
