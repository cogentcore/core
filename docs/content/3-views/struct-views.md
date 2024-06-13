# Forms

Cogent Core provides interactive forms that allow you to display a struct value to users and have them edit it.

You can make a form from any struct pointer:

```Go
type person struct {
    Name string
    Age  int
}
views.NewForm(parent).SetStruct(&person{Name: "Go", Age: 35})
```

You can make a form that fits in one line:

```Go
type person struct {
    Name string
    Age  int
}
views.NewForm(parent).SetInline(true).SetStruct(&person{Name: "Go", Age: 35})
```

You can detect when the user changes the value of the struct:

```Go
type person struct {
    Name string
    Age  int
}
p := person{Name: "Go", Age: 35}
views.NewForm(parent).SetStruct(&p).OnChange(func(e events.Event) {
    core.MessageSnackbar(parent, fmt.Sprintf("You are %v", p))
})
```

You can make it so that change events are sent immediately as the user types:

```Go
type person struct {
    Name string `immediate:"+"`
    Age  int
}
p := person{Name: "Go", Age: 35}
views.NewForm(parent).SetStruct(&p).OnChange(func(e events.Event) {
    core.MessageSnackbar(parent, fmt.Sprintf("You are %v", p))
})
```

You can hide certain fields from the user:

```Go
type person struct {
    Name string
    Age  int `view:"-"`
}
views.NewForm(parent).SetStruct(&person{Name: "Go", Age: 35})
```

You can prevent the user from editing certain fields:

```Go
type person struct {
    Name string `edit:"-"`
    Age  int
}
views.NewForm(parent).SetStruct(&person{Name: "Go", Age: 35})
```

You can prevent the user from editing the entire struct:

```Go
type person struct {
    Name string
    Age  int
}
views.NewForm(parent).SetStruct(&person{Name: "Go", Age: 35}).SetReadOnly(true)
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
views.NewForm(parent).SetStruct(&employee{Person{Name: "Go", Age: 35}, "Programmer"})
```

You can expand fields that are themselves structs:

```Go
type person struct {
    Name string
    Age  int
}
type employee struct {
    Role    string
    Manager person `view:"add-fields"`
}
views.NewForm(parent).SetStruct(&employee{"Programmer", person{Name: "Go", Age: 35}})
```

You can specify a default value (or list or range of values) for a field, which will be displayed in the tooltip for the field label, make the label highlighted when the value is non-default, and allow the user to reset the value to the default value by double clicking on the label:

```Go
type person struct {
    Name      string `default:"Gopher"`
    Age       int    `default:"20:30"`
    Precision int    `default:"64,32"`
}
views.NewForm(parent).SetStruct(&person{Name: "Go", Age: 35, Precision: 50})
```

You can make it so that the documentation comments for struct fields are used as tooltips for the field label and value widgets by adding the type to [[types]] and running `core generate`:

```go
// Add this once per package:
//go:generate core generate

// Add types:add for every type you want the documentation for:
type person struct { //types:add

    // This comment will be displayed in the tooltip for this field
    Name string
}
```

When you use [[core.NewValue]] with a struct value, it will create an inline form if the struct has four or fewer fields:

```Go
type person struct {
    Name string
    Age  int
}
// core.NewValue(&person{Name: "Go", Age: 35}, "", parent)
```

Otherwise, it will create a button that opens a dialog with a normal form:

```Go
type person struct {
    Name        string
    Age         int
    Job         string
    LikesGo     bool
    LikesPython bool
}
// core.NewValue(&person{Name: "Go", Age: 35, Job: "Programmer", LikesGo: true}, "", parent)
```
