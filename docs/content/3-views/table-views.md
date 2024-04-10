# Table views

Cogent Core provides interactive table views that allow you to display a slice of structs to users as a table and have them edit it. Users can use context menus and drag-and-drop to add, remove, and move rows. Also, users can sort the table by clicking on columns.

You can make a table view from any pointer to a slice of structs:

```Go
type language struct {
    Name   string
    Rating int
}
views.NewTableView(parent).SetSlice(&[]language{{"Go", 10}, {"Python", 5}})
```

You can detect when the user changes the value of the table:

```Go
type language struct {
    Name   string
    Rating int
}
sl := []language{{"Go", 10}, {"Python", 5}}
views.NewTableView(parent).SetSlice(&sl).OnChange(func(e events.Event) {
    core.MessageSnackbar(parent, fmt.Sprintf("Languages: %v", sl))
})
```

You can hide certain columns from the user:

```Go
type language struct {
    Name   string
    Rating int `view:"-"`
}
views.NewTableView(parent).SetSlice(&[]language{{"Go", 10}, {"Python", 5}})
```

You can also use the `tableview` struct tag, which overrides the `view` struct tag. This allows you to have a struct field displayed in a struct view but not a table view, or vise versa:

```Go
type language struct {
    Name   string
    Rating int `view:"-" tableview:"+"`
}
views.NewTableView(parent).SetSlice(&[]language{{"Go", 10}, {"Python", 5}})
```

You can prevent the user from editing certain columns:

```Go
type language struct {
    Name   string `edit:"-"`
    Rating int
}
views.NewTableView(parent).SetSlice(&[]language{{"Go", 10}, {"Python", 5}})
```

You can prevent the user from editing the entire table:

```Go
type language struct {
    Name   string
    Rating int
}
views.NewTableView(parent).SetSlice(&[]language{{"Go", 10}, {"Python", 5}}).SetReadOnly(true)
```

You can make it so that the documentation comments for struct fields are used as tooltips for the column headers and value widgets by adding the type to [[gti]] and running `core generate`:

```go
// Add this once per package:
//go:generate core generate

// Add gti:add for every type you want the documentation for:
type language struct { //gti:add

    // This comment will be displayed in the tooltip for this field
    Name string
}
```

When you use [[views.NewValue]] with a slice of structs, it will create a button that opens a dialog with a table view:

```Go
type language struct {
    Name   string
    Rating int
}
views.NewValue(parent, &[]language{{"Go", 10}, {"Python", 5}})
```
