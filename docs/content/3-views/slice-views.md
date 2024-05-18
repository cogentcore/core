# Slice views

Cogent Core provides interactive slice views that allow you to display a slice value to users and have them edit it. Users can use context menus and drag-and-drop to add, remove, and move rows.

You can make a slice view from any slice pointer:

```Go
views.NewSliceView(parent).SetSlice(&[]int{1, 3, 5})
```

You can make a slice view that fits in one line:

```Go
views.NewSliceViewInline(parent).SetSlice(&[]int{1, 3, 5})
```

You can detect when the user changes the value of the slice:

```Go
sl := []int{1, 3, 5}
views.NewSliceView(parent).SetSlice(&sl).OnChange(func(e events.Event) {
    core.MessageSnackbar(parent, fmt.Sprintf("Slice: %v", sl))
})
```

You can prevent the user from editing the slice:

```Go
views.NewSliceView(parent).SetSlice(&[]int{1, 3, 5}).SetReadOnly(true)
```

When you use [[views.NewValue]] with a slice value, it will create an inline slice view if the slice has four or fewer elements:

```Go
views.NewValue(parent, &[]int{1, 3, 5})
```

Otherwise, it will create a button that opens a dialog with a normal slice view:

```Go
views.NewValue(parent, &[]int{1, 3, 5, 7, 9})
```
