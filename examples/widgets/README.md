# GoGi Widgets Demo

This demo shows all the main widgets in GoGi.  You can try each of the widgets.  The most advanced features are available when you do `Control+Alt+I` or hit the button with the `Open GoGiEditor` on it.  Also try changing the Preferences using the option in the `widgets` "app" menu).

## Installation

The usual Go install procedure will work -- see [Install](https://github.com/goki/gi/wiki/Install) for details.

## GoGi Editor

This editor allows you to interactively build new GUI designs, and save / load them from JSON files.

The left panel shows the `TreeView` representation of the 2D scenegraph underlying the widgets demo.  You can click on the various nodes and see / edit all the properties in the right panel, which is showing a `StructView` property editor for the selected node.

This editor demonstrates the `ValueView` framework that manages the mapping of arbitrary values using the `reflect.Value` system, to editable GUI representations of those values, as then used in larger-scale views such as the `StructView`.  This editor is fully generic and available for editing any kind of content -- thus providing a basic default gui for interacting with native Go datastructures.

Here are some fun things to try:

* The `WinVp` node defines some `CSS` style properties.  you can edit the colors directly, or click on the various `Map: ki.Props` buttons to pull up editor dialogs, including a color configuration dialog using sliders (click on the colored "pencil" edit button after each set of RGB values).  Hit the `Update` button in the GoGi Editor for the changes to take effect in the original Widgets demo window.

* You can define all manner of different types of style settings -- try adding some new ones, and changing the type from `string` to some of the other appropriate types -- using specific types gives you a better editing experience -- e.g., by providing choosers for "enum" constant values.  These are all parsed directly from the Go code and made available through the `fmt.Stringer` kinds of mechanisms, augmented with the `kit.Enums` enum registry.

* Hit the `Update` button to cause your changes to be reflected in the original Widgets window -- some changes take effect immediately but others require a more extensive rebuild..

* Try clicking on the `edit1` node, toward the bpttom of the tree.  Then go over to the Widgets window and start editing in the "Enter text here..." text field -- you'll see the under-the-hood updates to the various fields for the widget, as you type!  This automatic updating is driven automatically by the fact that the tree node connects directly to its "source" widget and receives its update signals.

* Add some new widgets!  The context menu (right mouse button, two-finger click, etc) in the tree view has a menu for adding and deleting items.  You can choose the type to add, etc.  The types are all automatically registered through the `kit.Types` registry.

* Drag-n-drop items around in the tree view to move them around in the window -- holding the `Shift` key does a move, and otherwise it does a copy.

* Try making an SVG box, and adding some SVG elements to it.  You can enter a size in the `ViewBox` size field.  It is also a good idea to turn on the `Fill` toggle for the SVG box, and then set the `fill` property of the objects you add to be something different than white -- just click the add action on `Properties` enter fill / red or whatever.  You could also set the `min-width` and `min-height` properties on the SVG box to give it a specific size.

* Save your design, then close the app, and restart it -- you can do `Delete` on the `main-vlay` node just below `WinVp` (the main window viewport, which can not be deleted as it is the root of the hierarchy), and then do `Open` to see it reload.  By default it does the minimal changes when loading over an existing tree, so you can also try loading on top of the existing Widgets and see your changes get applied.

## Preferences

The `GoGi Preferences` menu item has may settings that you can change, including Key maps, color schemes, etc.  Hit the `Update` button to have changes apply to any open windows.

* Try `Colors/Open Colors` and choose one of the `ColorPrefs_*` files, then hit `Update` to see the different schemes.  

* You can select a Key Map -- to see the actual key mappings, do `Edit Key Maps` click on one of the maps.  If you toggle the `Sort` button you can see the items sorted either by the key or the functions. Click into one of the shortcut fields and enter a key sequence on your keyboard and it should show up there.  The context menu also lets you clear it.

## Zooming

Because everything is just rendered directly, it is easy to adjust the scale. Use the standard `Command +` or `-` keys to zoom in and out.  You can save a default zoom level and will in the future have support for different zoom levels -- the `for current screen only` option means that it will save the zoom level by the name of the current screen (e.g., if you have different external monitors for your laptop).

## Summary

The `ValueView` generic relfection-based gui supports all the above functionality with essentially no additional code.  Thus, the system is highly efficient and elegant, and leverages the power of the Go reflection system to minimize the additional coding necessary to interact with Go data structures.


