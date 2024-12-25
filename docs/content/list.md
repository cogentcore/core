+++
Categories = ["Widgets"]
+++

A **list** is a [[widget]] that allows you to display a slice value to users and have them edit it. Users can use [[context menu]]s and [[events#drag-and-drop]] to add, remove, and move rows.

For a slice of structs, use a [[table]] instead. For a single struct, use a [[form]]. For a map, use a [[keyed list]]. For a nested tree, use a [[tree]].

## Properties

You can make a list from any slice pointer:

```Go
core.NewList(b).SetSlice(&[]int{1, 3, 5})
```

You can make a list that fits in one line:

```Go
core.NewInlineList(b).SetSlice(&[]int{1, 3, 5})
```

You can prevent users from editing a list:

```Go
core.NewList(b).SetSlice(&[]int{1, 3, 5}).SetReadOnly(true)
```

## Events

You can detect when a user [[events#change]]s the value of a list:

```Go
sl := []int{1, 3, 5}
core.NewList(b).SetSlice(&sl).OnChange(func(e events.Event) {
    core.MessageSnackbar(b, fmt.Sprintf("Slice: %v", sl))
})
```

## List button

You can make a [[button]] that opens a [[dialog]] with a list:

```Go
core.NewListButton(b).SetSlice(&[]int{1, 3, 5})
```
