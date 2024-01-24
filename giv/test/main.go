package main

import (
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
)

func main() {
	b := gi.NewAppBody("treeTable")
	giv.TreeTable(b)
	b.RunMainWindow()
}
