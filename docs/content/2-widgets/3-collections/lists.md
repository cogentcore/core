# Lists

Cogent Core provides interactive lists that allow you to display a slice value to users and have them edit it. Users can use context menus and drag-and-drop to add, remove, and move rows.

You can make a list from any slice pointer:

```Go
core.NewList(b).SetSlice(&[]int{1, 3, 5})
```

You can make a list that fits in one line:

```Go
core.NewInlineList(b).SetSlice(&[]int{1, 3, 5})
```

You can detect when the user changes the value of the list:

```Go
sl := []int{1, 3, 5}
core.NewList(b).SetSlice(&sl).OnChange(func(e events.Event) {
    core.MessageSnackbar(b, fmt.Sprintf("Slice: %v", sl))
})
```

You can prevent the user from editing the list:

```Go
core.NewList(b).SetSlice(&[]int{1, 3, 5}).SetReadOnly(true)
```

You can make a button that opens a dialog with a list:

```Go
core.NewListButton(b).SetSlice(&[]int{1, 3, 5})
```
