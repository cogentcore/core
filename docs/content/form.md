+++
Categories = ["Widgets"]
+++

A **form** is a [[widget]] that allows you to display a struct value to users and have them edit it.

For a slice of structs, use a [[table]] instead. For a slice of non-structs, use a [[list]]. For a map, use a [[keyed list]]. For a nested tree, use a [[tree]].

## Properties

You can make a form from any struct pointer:

```Go
type person struct {
    Name string
    Age  int
}
core.NewForm(b).SetStruct(&person{Name: "Go", Age: 35})
```

You can make a form that fits in one line:

```Go
type person struct {
    Name string
    Age  int
}
core.NewForm(b).SetInline(true).SetStruct(&person{Name: "Go", Age: 35})
```

You can prevent users from editing a form:

```Go
type person struct {
    Name string
    Age  int
}
core.NewForm(b).SetStruct(&person{Name: "Go", Age: 35}).SetReadOnly(true)
```

## Events

You can detect when a user [[events#change]]s the value of a form:

```Go
type person struct {
    Name string
    Age  int
}
p := person{Name: "Go", Age: 35}
core.NewForm(b).SetStruct(&p).OnChange(func(e events.Event) {
    core.MessageSnackbar(b, fmt.Sprintf("You are %v", p))
})
```

You can make it so that change events are sent immediately as the user types (like [[events#input]]):

```Go
type person struct {
    Name string `immediate:"+"`
    Age  int
}
p := person{Name: "Go", Age: 35}
core.NewForm(b).SetStruct(&p).OnChange(func(e events.Event) {
    core.MessageSnackbar(b, fmt.Sprintf("You are %v", p))
})
```

## Struct tags

*See [[struct tags]] for a concise overview of all struct tags.*

You can hide certain fields from the user:

```Go
type person struct {
    Name string
    Age  int `display:"-"`
}
core.NewForm(b).SetStruct(&person{Name: "Go", Age: 35})
```

You can prevent the user from editing certain fields:

```Go
type person struct {
    Name string `edit:"-"`
    Age  int
}
core.NewForm(b).SetStruct(&person{Name: "Go", Age: 35})
```

You can change the label of a field:

```Go
type person struct {
    Name string `label:"Nickname"`
    Age  int
}
core.NewForm(b).SetStruct(&person{Name: "Go", Age: 35})
```

You can use structs with embedded fields:

```Go
type Person struct {
    Name string
    Age  int
}
type employee struct {
    Person
    Role string
}
core.NewForm(b).SetStruct(&employee{Person{Name: "Go", Age: 35}, "Programmer"})
```

You can display fields that are themselves structs:

```Go
type person struct {
    Name string
    Age  int
}
type employee struct {
    Role    string
    Manager person
}
core.NewForm(b).SetStruct(&employee{"Programmer", person{Name: "Go", Age: 35}})
```

You can expand fields that are themselves structs:

```Go
type person struct {
    Name string
    Age  int
}
type employee struct {
    Role    string
    Manager person `display:"add-fields"`
}
core.NewForm(b).SetStruct(&employee{"Programmer", person{Name: "Go", Age: 35}})
```

You can specify a default value (or list or range of values) for a field, which will be displayed in the [[tooltip]] for the field label [[text]], make the label highlighted when the value is non-default, and allow the user to reset the value to the default value by double clicking on the label:

```Go
type person struct {
    Name      string `default:"Gopher"`
    Age       int    `default:"20:30"`
    Precision int    `default:"64,32"`
}
core.NewForm(b).SetStruct(&person{Name: "Go", Age: 35, Precision: 50})
```

## Form button

You can make a [[button]] that opens a [[dialog]] with a form:

```Go
type person struct {
    Name string
    Age  int
}
core.NewFormButton(b).SetStruct(&person{Name: "Go", Age: 35})
```

## Generate

You can make it so that the documentation comments for struct fields are used as [[tooltip]]s for the label and value widgets of a form by adding the type to [[generate#types]] and running [[generate]]:

```go
// Add this once per package:
//go:generate core generate

// Add types:add for every type you want the documentation for:
type person struct { //types:add

    // This comment will be displayed in the tooltip for this field
    Name string
}
```
