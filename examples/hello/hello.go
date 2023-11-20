package main

import (
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/gimain"
)

func main() { gimain.Run(app) }

func app() {
	body := gi.NewBody()
	gi.NewLabel(body).SetText("Hello, World!")
	gi.NewWindowBody(body).Run().Wait() // makes a scene behind the scenes..

	gi.NewDialog(body).SetModal(true).
	
	gi.NewWindow(gi.NewScene(body)).Run().Wait()
	
	scene := gi.NewScene(body) <- sets parent
	scene.TopAppBar = ...
	body <- overflow
	scene.BottomAppBar


	
	// scene := gi.NewScene()
	// gi.NewLabel(scene).SetText("Hello, World!")
	// gi.NewWindow(scene).Run().Wait()
}
