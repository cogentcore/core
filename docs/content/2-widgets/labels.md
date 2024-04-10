# Labels
      
Cogent Core provides customizable and selectable labels that can display many kinds of text.

You can make a label that renders plain text:

```Go
core.NewLabel(parent).SetText("Hello, world!")
```

You can make a label that renders long text, which will automatically wrap by default:

```Go
core.NewLabel(parent).SetText("This is a very long sentence that demonstrates how label content will overflow onto multiple lines when the size of the label text exceeds the size of its surrounding container; labels are a customizable widget that Cogent Core provides, allowing you to display many kinds of text")
```

You can use HTML formatting in a label:

```Go
core.NewLabel(parent).SetText(`<b>You</b> can use <i>HTML</i> <u>formatting</u> inside of <b><i><u>Cogent Core</u></i></b> labels, including <span style="color:red;background-color:yellow">custom styling</span> and <a href="https://example.com">links</a>`)
```

You can use one of the 15 preset label types to customize the appearance of the label:

```Go
core.NewLabel(parent).SetType(core.LabelHeadlineMedium).SetText("Hello, world!")
```

You can also use a styler to further customize the appearance of the label:

```Go
core.NewLabel(parent).SetText("Hello,\n\tworld!").Style(func(s *styles.Style) {
    s.Font.Size.Dp(21)
    s.Font.Style = styles.Italic
    s.Text.WhiteSpace = styles.WhiteSpacePre
    s.Color = colors.C(colors.Scheme.Success.Base)
    s.Font.Family = string(core.AppearanceSettings.MonoFont)
})
```
