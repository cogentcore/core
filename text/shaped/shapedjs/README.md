# html canvas (js) text shaping

The challenge faced by this package is to support text shaping using the html `canvas` `measureText` function, and the `canvas` `fillText` function to actually render the text, in a way that supports the necessary functionality.

The basic layout problem is tricky because:

* In all platforms, formatted [rich.Text] must be rendered in _spans_ where the same font styling parameters apply. Multiple spans can (and often do) appear on the same line of text.

* The `fillText` and `measureText` functions only operate with a single active font style at a time (i.e., one span), and render requires an X, Y position to render at.

* Layout and line wrapping requires assembling these spans and breaking them up at the right spots, which requires splitting spans and thus the ability to measure smaller chunks of text to figure out where to split. Doing this the right way in an internationalized text context requires knowledge of graphemes, and re-shaping spans after splitting them, etc. There is lots of complex logic in go-text for doing all of this.

* Therefore, **we cannot rely on the canvas functions to do layout**. We could do some basic rune-based replication, but it wouldn't work for more complex languages.

* By far the best way forward is to get the font metric information into go-text and do everything there.

