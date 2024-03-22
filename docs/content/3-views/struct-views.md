# Struct views

Cogent Core provides interactive struct views that allow the user to view and edit struct values.

You can make a struct view from any struct:

```Go
type myStruct struct {
    Name string
    Age int
}
giv.NewStructView(parent).SetStruct(&myStruct{Name: "Go", Age: 35})
```