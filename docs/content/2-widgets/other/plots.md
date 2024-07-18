Cogent Core provides interactive and customizable data plots.

You can make an interactive plot from slice data:

```Go
type Data struct {
	Time       float32
	Population float32
	Distance   float32
}
data := []Data{
    {0, 500, 1520},
    {1, 800, 860},
    {2, 1600, 930},
    {3, 1400, 1282},
}
dt := errors.Log1(table.NewSliceTable(data))
pe := plotcore.NewPlotEditor(b).SetTable(dt)
```
