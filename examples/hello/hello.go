package main

import "goki.dev/gi/v2/gi"

func main() {
	b := gi.NewAppBody("Hello")
	gi.NewLabel(b).SetText("Hello, World!")
	b.NewWindow().Run().Wait()
}
