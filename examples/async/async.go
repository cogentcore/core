// Command async demonstrates async updating in Cogent Core.
package main

import (
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/icons"
)

type tableStruct struct {
	Icon       icons.Icon
	IntField   int
	FloatField float32
	StrField   string
	File       gi.Filename
}

const rows = 100000

func main() {
	table := make([]*tableStruct, 0, rows)
	b := gi.NewBody("Async Updating")
	tv := giv.NewTableView(b)
	tv.SetReadOnly(true)
	tv.SetSlice(&table)

	b.OnShow(func(e events.Event) {
		go func() {
			for i := 0; i < rows; i++ {
				updt := tv.UpdateStartAsync()
				table = append(table, &tableStruct{IntField: i, FloatField: float32(i) / 10.0})
				tv.UpdateWidgets()
				if len(table) > 0 {
					tv.ScrollToIdx(len(table) - 1)
				}
				tv.UpdateEndAsyncLayout(updt)
				time.Sleep(100 * time.Millisecond)
			}
		}()
	})

	b.RunMainWindow()
}
