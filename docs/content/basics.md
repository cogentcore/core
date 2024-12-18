The **basics** are a simple overview of the key [[concepts]] of Cogent Core. You can interactively run and edit the examples on this website directly, or you can [[install]] Cogent Core on your system and experiment locally. You can also use the [[playground]] to develop interactively. After you finish the basics, we recommend you read the [[tutorials]] and explore the [[widgets]].

## Hello world

This code makes a simple **hello world** example app:

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

## Apps


## Widgets


## Events


## Styling


## Updating


## Value binding


## Plans


## Async


## Next steps

Now that you understand the basics, you can apply them in [[tutorials]], explore the [[widgets]], or [[install]] Cogent Core on your system. You can also explore all of the [[concepts]] in greater depth.
