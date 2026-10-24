package main

import (
	"embed"

	"cogentcore.org/core/content"
	"cogentcore.org/core/core"
)

//go:embed content
var econtent embed.FS

func main() {
	b := core.NewBody("app")
	content.NewContent(b).SetContent(econtent)
	b.RunMainWindow()
}
