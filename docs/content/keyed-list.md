+++
Categories = ["Widgets"]
+++

A **keyed list** is a [[widget]] that allows you to display a map value to users and have them edit it. Users can use [[context menu]]s to add, remove, and move rows.

For a slice, use a [[list]] instead. For a slice of structs, use a [[table]]. For a single struct, use a [[form]]. For a nested tree, use a [[tree]].

## Properties

You can make a keyed list from any map pointer:

```Go
core.NewKeyedList(b).SetMap(&map[string]int{"Go": 1, "C++": 3, "Python": 5})
```

You can make a keyed list that fits in one line:

```Go
core.NewKeyedList(b).SetInline(true).SetMap(&map[string]int{"Go": 1, "C++": 3})
```

You can prevent users from editing a keyed list:

```Go
core.NewKeyedList(b).SetMap(&map[string]int{"Go": 1, "C++": 3, "Python": 5}).SetReadOnly(true)
```

You can make a keyed list with elements of any type:

```Go
core.NewKeyedList(b).SetMap(&map[string]any{"Go": 1, "C++": "C-like", "Python": true})
```

## Events

You can detect when a user [[events#change]]s the value of a keyed list:

```Go
m := map[string]int{"Go": 1, "C++": 3, "Python": 5}
core.NewKeyedList(b).SetMap(&m).OnChange(func(e events.Event) {
    core.MessageSnackbar(b, fmt.Sprintf("Map: %v", m))
})
```

## Keyed list button

You can make a [[button]] that opens a [[dialog]] with a keyed list:

```Go
core.NewKeyedListButton(b).SetMap(&map[string]int{"Go": 1, "C++": 3, "Python": 5})
```
