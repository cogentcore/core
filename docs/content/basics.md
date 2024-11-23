The **basics** are a simple overview of the key [[concepts]] of Cogent Core. We recommend you read the basics before the [[tutorials]] and [[install|install instructions]].

## Hello world

This code makes a simple hello world example app using Cogent Core:

```Go
package main

import "cogentcore.org/core/core"

func main() {
    b := core.NewBody()
    core.NewButton(b).SetText("Hello, World!")
    b.RunMainWindow()
}
```

Notice how you can see the result of the code above, a [[button]] with the [[text]] "Hello, World!". Not only can you see the result of the code, you can edit the code live. Try changing "Hello, World!" to "Click me!" and you will see the button update accordingly.

Even though Cogent Core is written in Go, a compiled language, it uses the interpreter [yaegi](https://github.com/cogentcore/yaegi) to provide interactive editing. You can edit almost all of the examples on this website and see the result immediately. You can also use the [[playground]] to experiment interactively with Cogent Core.

## Apps

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
