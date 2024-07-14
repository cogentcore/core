# Pages

Cogent Core provides [[pages]], a package for creating content-focused sites consisting of Markdown, HTML, and Cogent Core. This website that you are reading right now is made entirely using pages.

For example, this code recursively makes a copy of this website embedded into this page:

```Go
pages.NewPage(b).SetContent(content)
```
