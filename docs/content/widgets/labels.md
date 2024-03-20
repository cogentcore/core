# Labels
      
Cogent Core provides customizable labels that can display many kinds of text.

You can make a label that renders plain text:

```Go
gi.NewLabel(parent).SetText("Hello, world!")
```

You can make a label that renders long text, which will automatically wrap by default:

```Go
gi.NewLabel(parent).SetText("This is a very long sentence that demonstrates how label content will overflow onto multiple lines when the size of the label text exceeds the size of its surrounding container; labels are a customizable widget that Cogent Core provides, allowing you to display many kinds of text")
```
