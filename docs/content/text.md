+++
Categories = ["Widgets"]
+++

**Text** is a [[widget]] that can display many kinds of text.

You can render plain text:

```Go
core.NewText(b).SetText("Hello, world!")
```

You can render long text, which will automatically wrap by default:

```Go
core.NewText(b).SetText("This is a very long sentence that demonstrates how text content will overflow onto multiple lines when the size of the text exceeds the size of its surrounding container; text widgets are customizable widget that Cogent Core provides, allowing you to display many kinds of text")
```

You can use HTML formatting in text:

```Go
core.NewText(b).SetText(`<b>You</b> can use <i>HTML</i> <u>formatting</u> inside of <b><i><u>Cogent Core</u></i></b> text, including <span style="color:red;background-color:yellow">custom styling</span> and <a href="https://example.com">links</a>`)
```

## Styles

You can use one of the 15 preset text types to customize the appearance of text:

```Go
core.NewText(b).SetType(core.TextHeadlineMedium).SetText("Hello, world!")
```

You can also use a [[style]]r to further customize the appearance of text:

```Go
core.NewText(b).SetText("Hello,\n\tworld!").Styler(func(s *styles.Style) {
    s.Font.Size.Dp(21)
    s.Font.Slant = rich.Italic
    s.Font.SetDecoration(styles.Underline, styles.LineThrough)
    s.Text.WhiteSpace = styles.WhiteSpacePre
    s.Color = colors.Scheme.Success.Base
    s.SetMono(true)
})
```
