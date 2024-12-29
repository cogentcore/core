+++
Categories = ["Widgets"]
+++

A **file tree** is a [[widget]] that allow users to view directories as nested [[tree]]s.

For displaying a list of files with metadata, consider a [[file picker]] instead.

## Properties

You can make a file tree and open it at any filepath:

```Go
filetree.NewTree(b).OpenPath(".")
```

## Events

You can detect when a user selects files:

```Go
ft := filetree.NewTree(b).OpenPath(".")
ft.OnSelect(func(e events.Event) {
    selected := []string{}
    ft.SelectedFunc(func(n *filetree.Node) {
        selected = append(selected, string(n.Filepath))
    })
    core.MessageSnackbar(ft, strings.Join(selected, " "))
})
```

You can filter which files or directories are shown:

<!-- TODO: this is crashing yaegi -->
```go
ft := filetree.NewTree(b)
ft.FilterFunc = func(path string, info fs.FileInfo) bool {
    return info.IsDir() // only show directories
}
ft.OpenPath(".")
```
