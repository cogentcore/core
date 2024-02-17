package main

import (
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/icons"
)

type TableStruct struct { //gti:add
	Icon       icons.Icon
	IntField   int `default:"2"`
	FloatField float32
	StrField   string
	File       gi.Filename
}

func main() {
	table := make([]*TableStruct, 0, 100000)
	b := gi.NewBody("concurrent-map")
	tv := giv.NewTableView(b, "tv")
	tv.SetReadOnly(true)
	tv.SetSlice(&table)

	b.OnShow(func(e events.Event) {
		go func() {
			for i := 0; i < 100000; i++ {
				table = append(table, &TableStruct{IntField: i, FloatField: float32(i) / 10.0})
				updt := tv.UpdateStartAsync()
				tv.UpdateWidgets()
				tv.UpdateEndAsyncRender(updt)
				if len(table) > 0 {
					tv.ScrollToIdx(len(table) - 1)
				}
			}
		}()
	})

	b.RunMainWindow()
}
