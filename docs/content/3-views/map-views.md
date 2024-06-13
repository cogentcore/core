# Key value tables

Cogent Core provides interactive key value tables that allow you to display a map value to users and have them edit it. Users can use context menus to add, remove, and move rows.

You can make a key value table from any map pointer:

```Go
views.NewKeyValueTable(parent).SetMap(&map[string]int{"Go": 1, "C++": 3, "Python": 5})
```

You can make a key value table that fits in one line:

```Go
views.NewKeyValueTable(parent).SetInline(true).SetMap(&map[string]int{"Go": 1, "C++": 3})
```

You can detect when the user changes the value of the map:

```Go
m := map[string]int{"Go": 1, "C++": 3, "Python": 5}
views.NewKeyValueTable(parent).SetMap(&m).OnChange(func(e events.Event) {
    core.MessageSnackbar(parent, fmt.Sprintf("Map: %v", m))
})
```

You can prevent the user from editing the map:

```Go
views.NewKeyValueTable(parent).SetMap(&map[string]int{"Go": 1, "C++": 3, "Python": 5}).SetReadOnly(true)
```

You can make a key value table with elements of any type:

```Go
views.NewKeyValueTable(parent).SetMap(&map[string]any{"Go": 1, "C++": "C-like", "Python": true})
```

When you use [[views.NewValue]] with a map value, it will create an inline key value table if the map has two or fewer elements:

```Go
// views.NewValue(parent, &map[string]int{"Go": 1, "C++": 3})
```

Otherwise, it will create a button that opens a dialog with a normal key value table:

```Go
// views.NewValue(parent, &map[string]int{"Go": 1, "C++": 3, "Python": 5})
```
