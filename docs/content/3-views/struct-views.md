# Struct views

Cogent Core provides interactive struct views that allow the user to view and edit struct values.

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