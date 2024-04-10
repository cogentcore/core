// Command async demonstrates async updating in Cogent Core.
package main

import (
	"time"

	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/giv"
	"cogentcore.org/core/icons"
)

type tableStruct struct {
	Icon       icons.Icon
	IntField   int
	FloatField float32
	StrField   string
	File       core.Filename
}

const rows = 100000

func main() {
	table := make([]*tableStruct, 0, rows)
	b := core.NewBody("Async Updating")
	tv := giv.NewTableView(b)
	tv.SetReadOnly(true)
	tv.SetSlice(&table)

	b.OnShow(func(e events.Event) {
		go func() {
			for i := 0; i < rows; i++ {
				b.AsyncLock()
				table = append(table, &tableStruct{IntField: i, FloatField: float32(i) / 10.0})
				tv.Update()
				if len(table) > 0 {
					tv.ScrollToIndex(len(table) - 1)
				}
				b.AsyncUnlock()
				time.Sleep(1 * time.Millisecond)
			}
		}()
	})

	b.RunMainWindow()
}
