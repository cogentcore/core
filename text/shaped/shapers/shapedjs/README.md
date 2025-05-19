# html canvas (js) text shaping

The challenge faced by this package is to support text shaping using the html `canvas` `measureText` function, and the `canvas` `fillText` function to actually render the text, in a way that supports the necessary functionality.

The basic layout problem is tricky because:

* In all platforms, formatted [rich.Text] must be rendered in _spans_ where the same font styling parameters apply. Multiple spans can (and often do) appear on the same line of text.

* The `fillText` and `measureText` functions only operate with a single active font style at a time (i.e., one span), and render requires an X, Y position to render at.

* Layout and line wrapping requires assembling these spans and breaking them up at the right spots, which requires splitting spans and thus the ability to measure smaller chunks of text to figure out where to split. Doing this the right way in an internationalized text context requires knowledge of graphemes, and re-shaping spans after splitting them, etc. There is lots of complex logic in go-text for doing all of this.

* Thus, it would be so much better to use go-text to manage all the layout, but that is not possible because there is no way to actually determine the font being used:  [stackoverflow](https://stackoverflow.com/questions/7444451/how-to-get-the-actual-rendered-font-when-its-not-defined-in-css) nor is it possible to even get a list of fonts that might be used: [stackoverflow](https://stackoverflow.com/questions/53638179/supported-fonts-in-html5-canvas-text)

Fortunately, a simple strategy that works well in practice is to use go-text for first-pass layout and line wrapping, via the [shapedgt](../shapedgt) package, and then go back through with html measureText on the actual chunks produced by that, and adjust the sizes accordingly. This is done in the `AdjustOutput` function.


