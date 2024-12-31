**Styles** contains explanations of common [[style]] properties. You can also see the API documentation for an [exhaustive list](https://pkg.go.dev/cogentcore.org/core/styles#Style) of style properties. You can experiment with style properties in the [[style playground]].

## Color

Many style properties involve [[color]]s, which can be specified in several ways as documented on that page.

You can set the content color of [[text]] or an [[icon]]:

```Go
tx := core.NewText(b).SetText("Success")
tx.Styler(func(s *styles.Style) {
    s.Color = colors.Scheme.Success.Base
})
```
