package main

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/goosi/events"
	"goki.dev/icons"
)

func main() {
	b := gi.NewAppBody("Basic")
	bt := gi.NewButton(b).SetText("Open dialog").SetIcon(icons.OpenInNew)
	bt.OnClick(func(e events.Event) {
		gi.NewBody().AddTitle("Hello, world!").AddOkOnly().NewDialog(bt).Run()
	})
	b.NewWindow().Run().Wait()
}
