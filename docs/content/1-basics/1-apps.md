The first call in every Cogent Core app is [[doc:core.NewBody]]. This creates and returns a new [[doc:core.Body]], which is a container in which app content is placed. This takes an optional name, which is used for the title of the app/window/tab.

After calling [[doc:core.NewBody]], you add content to the [[doc:core.Body]] that was returned, which is typically given the local variable name `b` for body.

Then, after adding content to your body, you can create and start a window from it using [[doc:core.Body.RunMainWindow]].

Therefore, the standard structure of a Cogent Core app looks like this:

```Go
package main

import "cogentcore.org/core/core"

func main() {
	b := core.NewBody("App Name")
	// Add app content here
	b.RunMainWindow()
}
```

For most of the code examples on this website, we will omit the outer structure of the app so that you can focus on the app content.
