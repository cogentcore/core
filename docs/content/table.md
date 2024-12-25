+++
Categories = ["Widgets"]
+++

A **table** allows you to display a slice of structs to users as a table and have them edit it. Users can use [[context menu]]s and drag-and-drop to add, remove, and move rows. Also, users can sort the table by clicking on columns.

## Properties

You can make a table from any pointer to a slice of structs:

```Go
type language struct {
    Name   string
    Rating int
}
core.NewTable(b).SetSlice(&[]language{{"Go", 10}, {"Python", 5}})
```

You can prevent users from editing a table:

```Go
type language struct {
    Name   string
    Rating int
}
core.NewTable(b).SetSlice(&[]language{{"Go", 10}, {"Python", 5}}).SetReadOnly(true)
```

## Events

You can detect when a user [[events#change]]s the value of a table:

```Go
type language struct {
    Name   string
    Rating int
}
sl := []language{{"Go", 10}, {"Python", 5}}
core.NewTable(b).SetSlice(&sl).OnChange(func(e events.Event) {
    core.MessageSnackbar(b, fmt.Sprintf("Languages: %v", sl))
})
```

## Struct tags

*See [[struct tags]] for a concise overview of all struct tags.*

You can hide certain columns from the user:

```Go
type language struct {
    Name   string
    Rating int `display:"-"`
}
core.NewTable(b).SetSlice(&[]language{{"Go", 10}, {"Python", 5}})
```

You can also use the `table` struct tag, which overrides the `display` struct tag. This allows you to have a struct field displayed in a form but not a table, or vice versa:

```Go
type language struct {
    Name   string
    Rating int `display:"-" table:"+"`
}
core.NewTable(b).SetSlice(&[]language{{"Go", 10}, {"Python", 5}})
```

You can prevent the user from editing certain columns:

```Go
type language struct {
    Name   string `edit:"-"`
    Rating int
}
core.NewTable(b).SetSlice(&[]language{{"Go", 10}, {"Python", 5}})
```

You can change the label of a column:

```Go
type language struct {
    Name   string
    Rating int `label:"Score"`
}
core.NewTable(b).SetSlice(&[]language{{"Go", 10}, {"Python", 5}})
```

## List button

You can make a button that opens a dialog with a table:

```Go
type language struct {
    Name   string
    Rating int
}
core.NewListButton(b).SetSlice(&[]language{{"Go", 10}, {"Python", 5}})
```

## Generate

You can make it so that the documentation comments for struct fields are used as [[tooltip]]s for the column headers and value widgets of a table by adding the type to [[generate#types]] and running [[generate]]:

```go
// Add this once per package:
//go:generate core generate

// Add types:add for every type you want the documentation for:
type language struct { //types:add

    // This comment will be displayed in the tooltip for this field
    Name string
}
```
