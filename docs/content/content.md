+++
Categories = ["Widgets"]
+++

**Cogent Content** is a [[widget]] for creating content-focused sites consisting of Markdown, HTML, and Cogent Core. This website that you are reading right now is made entirely using content.

Code blocks marked as ```` ```Go ```` are interactive examples: the page shows their code in an editor and runs it, like the examples on this website. The `core` tool compiles them into the app when it builds it; see [[build#code blocks of content apps]].

See the API documentation for [[doc:content]] for more information.

<!-- TODO: stuck in infinite loop

For example, this code makes a copy of this website embedded into this page:

```Go
content.NewContent(b).SetContent(econtent)
``` -->
