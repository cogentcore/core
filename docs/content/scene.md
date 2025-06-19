+++
Categories = ["Widgets"]
+++

A **scene** is a [[frame]] widget that contains all of the [[widget]]s of a window or dialog. It is the root of a widget tree.

The central child of a scene is a [[body]], which contains the main app content. However, the scene also manages optional [[#bars]] on the sides.

A scene has the [[doc:paint.Painter]] that all widgets use to [[render]], and it manages the [[event]]s for all of its widgets.

A scene "takes place" on a [[stage]], which is where you can customize the options for windows and dialogs, such as sizing and positioning.

## Bars

The [[doc:core.Scene.Bars]] are functions for configuring optional [[widget]]s on any side surrounding the central [[body]] content.

The top bar is often used for a [[toolbar]]:

```go
b.AddTopBar(func(bar *Frame) {
	NewToolbar(bar).Maker(func(p *tree.Plan) {
        tree.Add(p, func(w *core.Button) {
            w.SetText("Build")
        })
	})
})
```

The bottom bar is often used in [[dialog]]s:

```Go
bt := core.NewButton(b).SetText("Dialog")
bt.OnClick(func(e events.Event) {
	d := core.NewBody("Dialog")
	d.AddBottomBar(func(bar *core.Frame) {
		d.AddCancel(bar)
		d.AddOK(bar)
	})
	d.RunDialog(bt)
})
```
