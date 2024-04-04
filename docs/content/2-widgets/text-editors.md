# Text editors

Cogent Core provides powerful text editors that support advanced code editing features, like syntax highlighting, completion, undo and redo, copy and paste, rectangular selection, and word, line, and page based navigation, selection, and deletion.

Text editors should mainly be used for editing code and other syntactic data like markdown and JSON. For simpler use cases, consider using text fields instead.

You can make a text editor without any custom options:

```Go
texteditor.NewSoloEditor(parent)
```

You can set the starting text of a text editor:

```Go
texteditor.NewSoloEditor(parent).Buffer.SetTextString("Hello, world!")
```

You can set the highlighting language of a text editor:

```Go
texteditor.NewSoloEditor(parent).Buffer.SetLang("go").SetTextString("package main")
```
