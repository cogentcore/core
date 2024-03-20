# Splits

Cogent Core provides customizable splits, which allow you to divide space among widgets and have the user customize how much space each widget gets using draggable handles.

You can make splits without any custom options:

```Go
sp := gi.NewSplits(parent)
gi.NewLabel(sp).SetText("First")
gi.NewLabel(sp).SetText("Last")
```