# GoGi Widgets Demo

This is the main demo.  You can try each of the widgets.  The most advanced features are available when you do `Control+Alt+E` or hit the button with the `GoGi Editor` on it.  Also try changing the preferences using `Control+Alt+P`.

## Installation

For Go newbies.. 

``` bash
> go get github.com/goki/goki
> cd ~/go/src/github.com/goki/goki/gi/examples/widgets
> go get ...
> go build
> ./widgets
```

**IMPORTANT for Linux users:** You need to install the Arial TTF font to get decent-looking rendering: https://askubuntu.com/questions/651441/how-to-install-arial-font-in-ubuntu.  Also, there is a known bug where closing windows exits the event loop!  You'll have to restart..

``` bash
> sudo apt-get install ttf-mscorefonts-installer
> sudo fc-cache
```

## GoGi Editor

This editor allows you to interactively build new GUI designs, and save / load them from JSON files.

The left panel shows the `TreeView` representation of the 2D scenegraph underlying the widgets demo.  You can click on the various nodes and see / edit all the properties in the right panel, which is showing a `StructView` property editor for the selected node.

This editor demonstrates the `ValueView` framework that manages the mapping of arbitrary values using the `reflect.Value` system, to editable GUI representations of those values, as then used in larger-scale views such as the `StructView`.  This editor is fully generic and available for editing any kind of content -- thus providing a basic default gui for interacting with native Go datastructures.

Here are some fun things to try:

* The `WinVp` node defines some `CSS` style properties.  you can edit the colors directly, or click on the various `...` buttons to pull up editor dialogs, including a color configuration dialog using sliders (that is the first ... button after each set of RGB values).

* You can define all manner of different types of style settings -- try adding some new ones, and changing the type from `string` to some of the other appropriate types -- using specific types gives you a better editing experience -- e.g., by providing choosers for "enum" constant values.  These are all parsed directly from the Go code and made available through the `fmt.Stringer` kinds of mechanisms, augmented with the `kit.Enums` enum registry.

* Hit the `Update` button to cause your changes to be reflected in the original Widgets window -- some changes take effect immediately but others require a more extensive rebuild..

* Try clicking on the `edit` node, toward the end of the tree.  Then go over to the Widgets window and start editing in the "Edit this text" text field -- you'll see the under-the-hood updates to the various fields for the widget, as you type!  This automatic updating is driven automatically by the fact that the tree node connects directly to its "source" widget and receives its update signals.

* Add some new widgets!  The `...` button in the tree view has a menu for adding and deleting items.  You can choose the type to add, etc.  The types are all automatically registered through the `kit.Types` registry.

* Try making an SVG box, and adding some SVG elements to it -- it is a good idea to turn on the `Fill` toggle for the SVG box, and then set the `fill` property of the objects you add to be something different than white -- just go to `Props/Edit Map` and `Add` and enter fill / red or whatever.  You'll need to set the `min-width` and `min-height` properties on the SVG box to give it a specific size..  As you can see, creating a drawing editor should be relatively easy from here..

* Save your design, then close the app, and restart it -- you can do `Delete` on the `vlay` node just below `WinVp` (the main window viewport, which can not be deleted as it is the root of the hierarchy), and then do Load JSON to see it reload.  By default it does the minimal changes when loading over an existing tree, so you can also try loading on top of the existing Widgets and see your changes get applied.

## Preferences

Hitting `Control+Alt+P` opens the preferences editor.  Hit the `Update` button to have changes apply.  You can easily change the default color scheme, and save the changes -- the prefs are automatically loaded for each app.

* Do `Default Key Map` and then you can edit the keymap and test out your changes -- you can see that the default map has emacs keys built in..

## Zooming

Because everything is just rendered directly, it is easy to adjust the scale. Use the standard `Command+` and `-` keys to zoom in and out.  You can save a default zoom level and will in the future have support for different zoom levels for different screens..

## Summary

The `ValueView` generic relfection-based gui supports all the above functionality with essentially no additional code.  Thus, the system is highly efficient and elegant, and leverages the power of the Go reflection system to minimize the additional coding necessary to interact with Go data structures.


