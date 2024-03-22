# Slice views

Cogent Core provides interactive slice views that allow you to display a slice value to the user and have them edit it. You can use the context menu to add, remove, and copy rows.

You can make a slice view from any slice pointer:

```Go
giv.NewSliceView(parent).SetSlice(&[]int{1, 3, 5})
```

You can detect when the user changes the value of the slice:

```Go
sl := []int{1, 3, 5}
giv.NewSliceView(parent).SetSlice(&sl).OnChange(func(e events.Event) {
    gi.MessageSnackbar(parent, fmt.Sprintf("Slice: %v", sl))
})
```
