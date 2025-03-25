# shapedgt: go-text implementation

This is the package that interfaces directly with go-text to turn rich text into _shaped_ text that is suitable for rendering. The lowest-level process is what harfbuzz does (https://harfbuzz.github.io/), shaping runes into combinations of glyphs. In more complex scripts, this can be a very complex process. In Latin languages like English, it is relatively straightforward. In any case, the result of shaping is a slice of `shaping.Output` representations, where each `Output` represents a `Run` of glyphs. Thus wrap the Output in a `Run` type, which adds more functions but uses the same underlying data.

The `Shaper` takes the rich text input and produces these elemental Run outputs. Then, the `WrapLines` function turns these runs into a sequence of `Line`s that sequence the runs into a full line.

One important feature of this shaping process is that _spaces_ are explicitly represented in the output.

