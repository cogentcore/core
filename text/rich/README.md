# Rich Text

The `rich.Text` type is the standard representation for formatted text, used as the input to the `shaped` package for text layout and rendering. It is encoded purely using `[]rune` slices for each span, with the style information represented with special rune values at the start of each span. This is an efficient and GPU-friendly pure-value format that avoids any issues of style struct pointer management etc.

It provides basic font styling properties (bold, italic, underline, font family, size, color) and some basic, essential broader formatting information.

It is physically a _flat_ format, with no hierarchical nesting of spans: any mechanism that creates `rich.Text` must compile the relevant nested spans down into a flat final representation, as the `htmltext` package does.

However, the `Specials` elements are indeed special, acting more like html start / end tags, using the special `End` value to end any previous special that was started (these must therefore be generated in a strictly nested manner). Use `StartSpecial` and `EndSpecial` to set these values safely, and helper methods for setting simple `Link`, `Super`, `Sub`, and `Math` spans are provided.

The `\n` newline is used to mark the end of a paragraph, and in general text will be automatically wrapped to fit a given size, in the `shaped` package. If the text starting after a newline has a ParagraphStart decoration, then it will be styled according to the `text.Style` paragraph styles (indent and paragraph spacing). The HTML parser sets this as appropriate based on `<br>` vs `<p>` tags.

