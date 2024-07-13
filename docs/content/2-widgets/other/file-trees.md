# File trees

Cogent Core provides powerful file trees that allow users to view directories as nested [trees](../collections/trees).

You can make a file tree and open it at any filepath:

```Go
filetree.NewTree(b).OpenPath(".")
```

You can detect when the user selects files:

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
