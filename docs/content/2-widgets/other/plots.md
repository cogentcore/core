Cogent Core provides interactive and customizable data plots.

You can make an interactive plot from slice data:

```Go
type Data struct {
	Time   float32
	Users  float32
	Profit float32
}
plotcore.NewPlotEditor(b).SetSlice([]Data{
    {0, 500, 1520},
    {1, 800, 860},
    {2, 1600, 930},
    {3, 1400, 682},
})
```
