# Slice views

Cogent Core provides interactive slice views that allow you to display a slice value to the user and have them edit it.

You can make a slice view from any slice pointer:

```Go
giv.NewSliceView(parent).SetSlice(&[]int{3, 5})
```