+++
Name = "HTML"
Categories = ["Widgets"]
+++

Cogent Core provides functions for converting HTML and Markdown into interactive [[widget]]s, with the [[doc:htmlcore]] package. HTML is styled with CSS as in browsers (style sheets, selectors, and the browser's default styles) and laid out with the [[layout]] of Cogent Core, which supports most of what pages use (flex and grid layouts, tables, floats, positioned elements, media and container queries). Not all of HTML and CSS are supported, and some things may render differently than in browsers.

## HTML

You can convert an HTML string into widgets:

```Go
htmlcore.ReadHTMLString(htmlcore.NewContext(), b, `<h1>Hello</h1><button>Click me!</button>`)
```

Style sheets and `style` attributes apply as in browsers:

```Go
htmlcore.ReadHTMLString(htmlcore.NewContext(), b, `<style>
.card { border: 1px solid gray; border-radius: 8px; padding: 8px 16px; display: flex; gap: 16px }
.card b { color: teal }
</style>
<div class="card"><b>Styled</b><span>with CSS</span></div>`)
```

The [[doc:htmlcore.Context]] has the settings of reading, and handlers for the widgets that are made, as for adding behavior to them:

```Go
ctx := htmlcore.NewContext()
ctx.AddWidgetHandler(func(w core.Widget) {
    if bt, ok := w.(*core.Button); ok {
        bt.OnClick(func(e events.Event) {
            core.MessageSnackbar(bt, "Clicked "+bt.Text)
        })
    }
})
htmlcore.ReadHTMLString(ctx, b, `<button>Click me!</button>`)
```

You can also read a parsed document with [[doc:htmlcore.ReadDocument]], from [[doc:htmlcore.ParseHTML]], and find the element of a widget with [[doc:htmlcore.NodeOf]]. For the pages of an app, set the URL of the page with [[doc:htmlcore.Context.SetPageURL]] (for relative URLs), and how to get resources and open links with [[doc:htmlcore.Context.SetGetURL]] and [[doc:htmlcore.Context.SetOpenURL]]. Custom elements can be made into widgets with [[doc:htmlcore.Context.AddElementHandler]].

## Web pages

The [[doc:htmlcore]] package is a small API over the engine that reads HTML, the [[doc:htmlcore/htmlengine]] package, which apps that show whole web pages, as a browser does, use directly. Its [[doc:htmlcore/htmlengine.Context]] has all of the settings of reading pages (as media sizes, scripting and zoom), and a new context is used for each page, with [[doc:htmlcore/htmlengine.Context.NewPage]]:

```go
ctx := base.NewPage()
ctx.PageURL = "https://example.com/"
resp, err := ctx.GetURL(ctx.PageURL)
...
doc, err := htmlengine.ParseHTML(ctx, resp.Body)
err = htmlengine.ReadDocument(ctx, page, doc)
```

When a document changes after it is read, as by the scripts of a page, [[doc:htmlcore/htmlengine.Context.Restyle]] updates the widgets for changes of the styles of elements (their `class` and `style`, and states such as `:hover`), and [[doc:htmlcore/htmlengine.Context.Reread]] reads only the elements that have changed again, keeping the widgets of the others. Both return false when the page needs to be read again whole. The web browser of [Cogent Web](https://github.com/cogentcore/cogent/tree/main/web) runs the scripts of pages with these.

## Markdown

You can convert a Markdown string into widgets:

```Go
htmlcore.ReadMDString(htmlcore.NewContext(), b, `# Hello
**This** is *some* ***Markdown*** [text](https://example.com)`)
```

Markdown is converted to HTML ([[doc:htmlcore.MDToHTML]]), which is then read as above.

## Implementation

This is a broad overview of how [[doc:htmlcore/htmlengine]] works; see the documentation of each part for more. The parts of CSS that do not depend on widgets (parsing style sheets, selectors, and the values of properties) are in [[doc:htmlcore/css]].

* **Parsing:** HTML is parsed into a document of nodes by [golang.org/x/net/html](https://pkg.go.dev/golang.org/x/net/html), as browsers parse it ([[doc:htmlcore/htmlengine.ParseHTML]]).

* **Cascade:** the style sheets of the document (`<style>`, `<link>`, `@import`, with `@media`, `@supports`, `@container` and `@layer`) and the default styles of browsers are matched against its elements with [cascadia](https://pkg.go.dev/github.com/andybalholm/cascadia) selectors, giving each element its declarations in the order of the cascade (by origin, `!important`, specificity and order), with `var()` resolved. [[doc:htmlcore/htmlengine.Context.ComputedValue]] returns the value of a property of an element.

* **Reading:** [[doc:htmlcore/htmlengine.ReadDocument]] walks the document, making the widgets of its elements in the widget given to it: block elements are [[frame]]s, runs of inline content (text, links, `<b>`, `<span>`) are single [[text]]s with rich text, and replaced elements and controls are their widgets ([[image]], [[svg]], [[button]], [[text field]], and so on). Element handlers ([[doc:htmlcore.Context.AddElementHandler]]) can make the widgets of elements themselves.

* **Styling:** the declarations of each element are compiled into a [[style|styler]] of its widget, which sets its [[doc:styles.Style]] from them when it is styled, with CSS units, inheritance and percentages resolved then, as the sizes of their containers are known.

* **Layout:** pages are laid out by the [[layout]] of Cogent Core, which has the intrinsic (min-content and preferred) widths, floats, positioning, grid placement and stacking that CSS needs. Tables and grids are arranged into grid frames as they are read, and multiple columns into column frames.

* **Updates:** [[doc:htmlcore/htmlengine.Context.Restyle]] and [[doc:htmlcore/htmlengine.Context.Reread]] find the declarations of the changed elements again from the recorded style rules, and update their widgets in place, reconciling the new widgets of reread elements with the old ones by [[plan|tree.Plan]].
