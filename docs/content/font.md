+++
Categories = ["Concepts"]
+++

Apps come with the default **fonts** of [Noto Sans](https://fonts.google.com/noto/specimen/Noto+Sans) and [Roboto Mono](https://fonts.google.com/specimen/Roboto+Mono). Apps automatically use system fonts for international text and emojis, so all characters are rendered in some font if possible. You can add custom fonts to your app on all platforms.

## Custom fonts

### Desktop and mobile

To add a custom font to your app, you need one or more `.ttf` (TrueType Font) files. We currently do *not* support variable fonts, although we [plan to](https://github.com/go-text/typesetting/issues/151) at some point.

1. Place all of your `.ttf` files in a new folder in your app directory called `myfontname` (ex: `opensans`).

2. Make a new file in your app directory called `myfontname.go` and add this code, replacing `myfontname` with your font name:

```go
//go:build !js

package main

import (
    "embed"

    "cogentcore.org/core/text/fonts"
)

//go:embed myfontname/*.ttf
var MyFontName embed.FS

func init() {
    fonts.AddEmbeddedFonts(MyFontName)
}
```

### Web

1. Find or make a CSS file that includes your font.

If your font is available on [Google Fonts](https://fonts.google.com/), you can click on `Get font` from the font's page, and then click on `Get embed code`. In the `Web` section under `Embed code in the &lt;head&gt; of your html`, you can find the download URL for the font as the `href` attribute of the last `&lt;link&gt;` element. Copy that URL. It should look something like this: `https://fonts.googleapis.com/css2?family=Noto+Sans:ital,wght@0,100..900;1,100..900&display=swap`

Otherwise, if your font is not on Google Fonts, you can make a CSS file with [font-face](https://developer.mozilla.org/en-US/docs/Web/CSS/@font-face) to load the font. Host that CSS file on your website or somewhere else and then copy its URL (it can be relative or absolute).

2. If you don't already have one, make a `core.toml` file in your app directory.

3. Add this somewhere in your `core.toml` file:

```toml
[Web]
  Styles = ["the_url_that_you_copied_in_step_one"]
```

### Web metrics

To get more accurate text layout on web for your custom font, you can embed the font metrics. This is not always necessary, and you only need to do this if your text is getting positioned strangely.

**Note:** If your font has a small file size or load time is not an issue, you can instead just include the normal embedded fonts for [[#desktop and mobile]] on web as well (by removing the `//go:build !js` build tag), but it is more optimal to do the web metrics approach.

1. Install the `metricsonly` tool:

```sh
go install cogentcore.org/core/text/fonts/metricsonly@main
```

2. Run `metricsonly` on your `myfontname` directory:

```sh
metricsonly myfontname/* -o myfontnamejs
```

For example:

```sh
metricsonly opensans/* -o opensansjs
```

3. Make a new file in your app directory called `myfontname_js.go` and add this code, replacing `myfontname` with your font name:

```go
//go:build js

package main

import (
    "embed"

    "cogentcore.org/core/text/fonts"
)

//go:embed myfontnamejs/*.ttf
var MyFontNameJS embed.FS

func init() {
    fonts.AddEmbeddedFonts(MyFontNameJS)
}
```
