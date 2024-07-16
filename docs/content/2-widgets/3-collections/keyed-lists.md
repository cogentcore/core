Cogent Core provides interactive keyed lists that allow you to display a map value to users and have them edit it. Users can use context menus to add, remove, and move rows.

You can make a keyed list from any map pointer:

```Go
core.NewKeyedList(b).SetMap(&map[string]int{"Go": 1, "C++": 3, "Python": 5})
```

You can make a keyed list that fits in one line:

```Go
core.NewKeyedList(b).SetInline(true).SetMap(&map[string]int{"Go": 1, "C++": 3})
```

You can detect when the user changes the value of the keyed list:

```Go
m := map[string]int{"Go": 1, "C++": 3, "Python": 5}
core.NewKeyedList(b).SetMap(&m).OnChange(func(e events.Event) {
    core.MessageSnackbar(b, fmt.Sprintf("Map: %v", m))
})
```

You can prevent the user from editing the keyed list:

```Go
core.NewKeyedList(b).SetMap(&map[string]int{"Go": 1, "C++": 3, "Python": 5}).SetReadOnly(true)
```

You can make a keyed list with elements of any type:

```Go
core.NewKeyedList(b).SetMap(&map[string]any{"Go": 1, "C++": "C-like", "Python": true})
```

You can make a button that opens a dialog with a keyed list:

```Go
core.NewKeyedListButton(b).SetMap(&map[string]int{"Go": 1, "C++": 3, "Python": 5})
```
