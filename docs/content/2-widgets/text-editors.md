# Text editors

Cogent Core provides powerful text editors that support advanced code editing features, like syntax highlighting, completion, undo and redo, copy and paste, rectangular selection, and word, line, and page based navigation, selection, and deletion.

Text editors should mainly be used for editing code and other syntactic data like markdown and JSON. For simpler use cases, consider using text fields instead.

You can make a text editor without any custom options:

```Go
texteditor.NewSoloEditor(parent)
```

You can set the text of a text editor:

```Go
texteditor.NewSoloEditor(parent).Buffer.SetTextString("Hello, world!")
```

You can set the highlighting language of a text editor:

```Go
texteditor.NewSoloEditor(parent).Buffer.SetLang("go").SetTextString(`package main

func main() {
    fmt.Println("Hello, world!")
}
`)
```

You can set the text of a text editor from an embedded file:

```go
//go:embed file.go
var myFile embed.FS
```

```Go
grr.Log(texteditor.NewSoloEditor(parent).Buffer.OpenFS(myFile, "file.go"))
```

You can also set the text of a text editor directly from the system filesystem, but this is not recommended for files built into your app, since they will end up in a different location on different platforms:

```go
grr.Log(texteditor.NewSoloEditor(parent).Buffer.Open("file.go"))
```
