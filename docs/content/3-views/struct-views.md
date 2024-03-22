# Struct views

Cogent Core provides interactive struct views that allow you to display a struct value to the user and have them edit it.

You can make a struct view from any struct:

```Go
type person struct {
    Name string
    Age int
}
giv.NewStructView(parent).SetStruct(&person{Name: "Go", Age: 35})
```

You can detect when the user changes the value of the struct:

```Go
type person struct {
    Name string
    Age int
}
p := person{Name: "Go", Age: 35}
giv.NewStructView(parent).SetStruct(&p).OnChange(func(e events.Event) {
    gi.MessageSnackbar(parent, fmt.Sprintf("You are %v", p))
})
```

You can make it so that change events are sent immediately as the user types:

```Go
type person struct {
    Name string `immediate:"+"`
    Age int
}
p := person{Name: "Go", Age: 35}
giv.NewStructView(parent).SetStruct(&p).OnChange(func(e events.Event) {
    gi.MessageSnackbar(parent, fmt.Sprintf("You are %v", p))
})
```

You can hide certain fields from the user:

```Go
type person struct {
    Name string
    Age int `view:"-"`
}
giv.NewStructView(parent).SetStruct(&person{Name: "Go", Age: 35})
```

You can prevent the user from editing certain fields:

```Go
type person struct {
    Name string `edit:"-"`
    Age int
}
giv.NewStructView(parent).SetStruct(&person{Name: "Go", Age: 35})
```

You can prevent the user from editing the entire struct:

```Go
type person struct {
    Name string
    Age int
}
giv.NewStructView(parent).SetStruct(&person{Name: "Go", Age: 35}).SetReadOnly(true)
```

You can specify a default value (or list or range of values) for a field, which will be displayed in the tooltip for the field label, make the label highlighted when the value is non-default, and allow the user to reset the value to the default value by double clicking on the label:

```Go
type person struct {
    Name string `default:"Gopher"`
    Age int `default:"20:30"`
    Precision int `default:"64,32"`
}
giv.NewStructView(parent).SetStruct(&person{Name: "Go", Age: 35, Precision: 50})
```
