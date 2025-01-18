+++
Name = "HTML"
Categories = ["Widgets"]
+++

Cogent Core provides functions for converting HTML and Markdown into interactive [[widget]]s. Not all of HTML and Markdown are supported yet, and some things may render incorrectly.

## HTML

You can convert an HTML string into widgets:

```Go
htmlcore.ReadHTMLString(htmlcore.NewContext(), b, `<h1>Hello</h1><button>Click me!</button>`)
```

## Markdown

You can convert a Markdown string into widgets:

```Go
htmlcore.ReadMDString(htmlcore.NewContext(), b, `# Hello
**This** is *some* ***Markdown*** [text](https://example.com)`)
```
