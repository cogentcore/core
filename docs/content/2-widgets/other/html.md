# HTML

Cogent Core provides functions for converting HTML and Markdown into interactive Cogent Core widgets. Not all of HTML and Markdown are supported yet, and some things may render incorrectly.

You can convert an HTML string into Cogent Core widgets:

```Go
htmlcore.ReadHTMLString(htmlcore.NewContext(), parent, `<h1>Hello</h1><button>Click me!</button>`)
```

You can convert a Markdown string into Cogent Core widgets:

```Go
htmlcore.ReadMDString(htmlcore.NewContext(), parent, `# Hello
**This** is *some* ***Markdown*** [text](https://example.com)`)
```
