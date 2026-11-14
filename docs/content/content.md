+++
Categories = ["Widgets"]
+++

**Cogent Content** is a [[widget]] for creating content-focused sites consisting of Markdown, HTML, and Cogent Core. This website that you are reading right now is made entirely using content.

## Sites made with content

Content scales to large sites, such as textbooks with hundreds of pages of figures, equations, citations, and interactive simulations. The source of these sites is a good place to see how its features are used:

* [compcogneuro.org](https://compcogneuro.org) ([source](https://github.com/compcogneuro/web)): the Computational Cognitive Neuroscience textbook, with interactive neural network simulations embedded in its pages.
* [wavereality.org](https://wavereality.org) ([source](https://github.com/WaveReality/web)): a site on physics, with a lot of math.
* [randalloreilly.com](https://randalloreilly.com) ([source](https://github.com/rcoreilly/web)): a personal website, built as static HTML pages without the app (see [[#HTML only sites]]).

## Making a site

A content site is an app whose main package embeds a directory of Markdown files and makes a [[doc:content.Content]] widget for them:

```go
//go:embed content
var econtent embed.FS

func main() {
	b := core.NewBody("My site")
	ct := content.NewContent(b).SetContent(econtent)
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(ct.MakeToolbar)
	})
	b.RunMainWindow()
}
```

Each Markdown file is a page. Its name is the name of the file in sentence case (`text-fields.md` is "Text fields") and its URL is the name in kebab case (`text-fields`). The page with the URL `""` (set in its front matter) is the home page.

A page can start with TOML front matter between `+++` lines, which sets fields of [[doc:content/bcontent.Page]]:

```toml
+++
Name = "Text fields"
Title = "Text fields and their uses"
Categories = ["Widgets"]
Authors = "Jane Smith"
Date = "2026-11-13"
+++
```

The categories make the categories tree in the sidebar. `Draft = true` keeps a page out of the website.

## Links

Pages link to each other with wikilinks: `[[text fields]]` links to the page with that name, `[[text fields#Styles]]` to a heading in it, and `[[text fields|fields of text]]` sets the text of the link. Other kinds of wikilinks can be added with `ct.Context.AddWikilinkHandler`; for example, this website adds `htmlcore.GoDocWikilink("doc", "cogentcore.org/core")`, so that `[[doc:core.Button]]` links to the API documentation of a Go symbol.

## Figures, tables, and equations

A line of attributes in braces before a block sets attributes of it. An `id` that starts with `figure_`, `table_`, `eq_`, or another name followed by an underscore makes a numbered element, which a wikilink such as `[[#figure_neuron]]` (or `[[neuron#figure_neuron]]` in another page) refers to as "Figure 1":

```md
{id="figure_neuron" style="height:20em"}
![A neuron, with its dendrites and axon.](media/neuron.png)

{id="eq_energy" title="Total energy"}
$$
E = \frac{1}{2} m v^2 + V(x)
$$
```

The alt text of an image is its caption. Math is written in LaTeX between `$` (inline) or `$$` (display), and rendered with the built-in TeX typesetting.

## Citations

With a list of references in [CSL JSON](https://citationstyles.org) format (as exported by Zotero), set as the `References` of the content, `[[@SmithJones89]]` cites a source as "Smith & Jones, 1989" and `[[@^SmithJones89]]` as "Smith & Jones (1989)", linking to the references page. The `mdcite` tool (`cogentcore.org/core/text/csl/mdcite`) makes a file of only the references that the pages cite.

## Code blocks

Code blocks marked as ```` ```Go ```` are interactive examples: the page shows their code in an editor and runs it, like the examples on this website. The `core` tool compiles them into the app when it builds it; see [[build#code blocks of content apps]].

## HTML only sites

`core build wasm -htmlonly` builds a site of static HTML pages, one for each page, without the app, with navigation trees and search like the app. These are the same pages that are shown while the app is loading (see [[build]]); interactive code blocks are not run. [randalloreilly.com](https://randalloreilly.com) is built this way.

`MakeToolbarPDF` adds a button that makes a PDF of a page, using the authors, date, and abstract in its front matter.

See the API documentation for [[doc:content]] for more information.

<!-- TODO: stuck in infinite loop

For example, this code makes a copy of this website embedded into this page:

```Go
content.NewContent(b).SetContent(econtent)
``` -->
