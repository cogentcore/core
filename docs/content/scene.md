+++
Categories = ["Architecture"]
+++

The **scene** contains all of the [[widget]] elements for a window or a dialog.

It is a type of [[frame]] with optional `Bars` elements that can be configured on each side around a central [[body]] element that has the main scene contents.

The Scene has the [[doc:paint.Painter]] that all widgets use to [[render]], and it manages the [[events]] for all of its widgets.

## Bars

The `Bars` on the scene contain functions for configuring optional [[toolbar]]-like elements on any side surrounding the central body content.

The `Top` bar is most frequently used, typically in this way:

```go
	b.AddTopBar(func(bar *Frame) {
		NewToolbar(bar).Maker(w.MakeToolbar)
	})
```

Where `w.MakeToolbar` is a [[plan#Maker function]] taking a `*tree.Plan` argument, that configures the [[toolbar]] with the [[button]] and [[func button]] actions for this scene.

## Stages

The [[doc:core.Stage]] provides the outer "setting" for a scene, and manages its behavior in relation to other scenes within the overall [[app]].

The different [[doc:core.StageTypes]] include things like `Window`, `Dialog`, `Menu`, `Tooltip` etc.

The `Window` and `Dialog` are _main_ stages, whereas the others are _popup_ stages, with different overall behavior.



