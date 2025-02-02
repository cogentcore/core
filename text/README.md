# text

This directory contains all of the text processing and rendering functionality, organized as follows.

## Sources

* `string`, `[]byte`, `[]rune` -- basic Go level representations of _source_ text, which can include `\n` `\r` line breaks, all manner of unicode characters, and require a language and script context to properly interpret.

* HTML or other _rich text_ formats (e.g., PDF, RTF, even .docx etc), which can include local text styling (bold, underline, font size, font family, etc), links, _and_ more complex, larger-scale elements including paragraphs, images, tables, etc.

## Levels:

* `Spans` or `Runs`: this is the smallest chunk of text above the individual runes, where all the runes share the same font, language, script etc characteristics. This is the level at which harfbuzz operates, transforming `Input` spans into `Output` runs.

* `Lines`: for line-based uses (e.g., texteditor), spans can be organized (strictly) into lines. This imposes strict LTR, RTL horizontal ordering, and greatly simplifies the layout process. Only text is relevant.

* `Text`: for otherwise unconstrained text rendering, you can have horizontal or vertical text that requires a potentially complex _layout_ process. `go-text` includes a `segmenter` for finding unicode-based units where line breaks might occur, and a `shaping.LineWrapper` that manages the basic line wrapping process using the unicode segments. `canvas` includes a `RichText` representation that supports Donald Knuth's linebreaking algorithm, which is used in LaTeX, and generally produces very nice looking results. This `RichText` also supports any arbitrary graphical element, so you get full layout of images along with text etc.

## Uses:

*  `texteditor.Editor`, planned `Terminal`: just need pure text, line-oriented results. This is the easy path and we don't need to discuss further.  Can use our new rich text span element instead of managing html for the highlighting / markup rendering.

* `core.Text`, `core.TextField`: pure text (no images) but ideally supports full arbitrary text layout. The overall layout engine is the core widget layout system, optimized for GUI-level layout, and in general there are challenges to integrating the text layout with this GUI layout, due to bidirectional constraints (text shape changes based on how much area it has, and how much area it has influences the overall widget layout). Knuth's algorithm explicitly handles the interdependencies through a dynamic programming approach.

* `svg.Text`: similar to core.Text but also requires arbitrary rotation and scaling parameters in the output, in addition to arbitrary x,y locations per-glyph that can be transformed overall.

* `htmlcore` and `content`: ideally would support LaTeX quality full rich text layout that includes images, "div" level grouping structures, tables, etc.

## Organization:

* `text/rich`: the `rich.Spans` is a `[][]rune` type that encodes the local font-level styling properties (bold, underline, etc) for the basic chunks of text _input_. This is the basic engine for basic harfbuzz shaping and all text rendering, and produces a corresponding `text/ptext` `ptext.Runs` _output_ that mirrors the Spans input and handles the basic machinery of text rendering. This is the replacement for the `ptext.Text`, `Span` and `Rune` elements that we have now.

* `text/lines`: manages Spans and Runs for line-oriented uses (texteditor, terminal). Need to move `parse/lexer/Pos` into lines, along with probably some of the other stuff from lexer, and move `parser/tokens` into `text/tokens` as it is needed to be our fully general token library for all markup. Probably just move parse under text too?

* `text/text`: manages the general purpose text layout framework. TODO: do we make the most general-purpose LaTeX layout system with arbitrary textobject elements as in canvas? Is this just the `core.Text` guy? textobjects are just wrappers around `render.Render` items -- need an interface that gives the size of the elements, and how much detail does the layout algorithm need?


