# lines

The lines package manages multi-line monospaced text with a given line width in runes, so that all text wrapping, editing, and navigation logic can be managed purely in text space, allowing rendering and GUI layout to be relatively fast.

This is suitable for text editing and terminal applications, among others. The text is encoded as runes along with a corresponding [rich.Text] markup representation with syntax highlighting, using either chroma or the [parse](../parse) package where available. A subsequent update will add support for the [gopls](https://pkg.go.dev/golang.org/x/tools/gopls) system and LSP more generally. The markup is updated in a separate goroutine for efficiency.

Everything is protected by an overall `sync.Mutex` and is safe to concurrent access, and thus nothing is exported and all access is through protected accessor functions. In general, all unexported methods do NOT lock, and all exported methods do.

## Views

Multiple different views onto the same underlying text content are supported, through the unexported `view` type. Each view can have a different width of characters in its formatting, which is the extent of formatting support for the view: it just manages line wrapping and maintains the total number of display lines (wrapped lines). The `Lines` object manages its own views directly, to ensure everything is updated when the content changes, with a unique ID (int) assigned to each view, which is passed with all view-related methods.

A widget will get its own view via the `NewView` method, and use `SetWidth` to update the view width accordingly (no problem to call even when no change in width). See the [textcore](../textcore) `Editor` for a base widget implementation.

## Events

Two standard events are sent by the `Lines`:
* `events.Input` (use `OnInput` to register a function to receive) is sent for every edit large or small.
* `events.Change` (`OnChange`) is sent for major changes: new text, opening files, saving files, `EditDone`.

Widgets should listen to these to update rendering and send their own events.

## Files

Full support for a file associated with the text lines is engaged by calling `SetFilename`. This will then cause it to check if the file has been modified prior to making any changes, and to save an autosave file (in a separate goroutine) after modifications, if `SetAutosave` is set.  Otherwise, no such file-related behavior occurs.

## Syntax highlighting

Syntax highlighting depends on detecting the type of text represented. This happens automatically via SetFilename, but must also be triggered using ?? TODO.

## Editing

* `InsertText`, `DeleteText` and `ReplaceText` are the core editing functions.
* `InsertTextRect` and `DeleteTextRect` support insert and delete on rectangular region defined by upper left and lower right coordinates, instead of a contiguous region.

All editing functionality uses [textpos](../textpos) types including `Pos`, `Region`, and `Edit`, which are based on the logical `Line`, `Char` coordinates of a rune in the original source text. For example, these are the indexes into `lines[pos.Line][pos.Char]`. In general, everything remains in these logical source coordinates, and the navigation functions (below) convert back and forth from these to the wrapped display layout, but this display layout is not really exposed.

## Undo / Redo

Every edit generates a `textpos.Edit` record, which is recorded by the undo system (if it is turned on, via `SetUndoOn` -- on by default). The `Undo` and `Redo` methods thus undo and redo these edits. The `NewUndoGroup` method is important for grouping edits into groups that will then be done all together, so a bunch of small edits are not painful to undo / redo.

The `Settings` parameters has an `EmacsUndo` option which adds undos to the undo record, so you can get fully nested undo / redo functionality, as is standard in emacs.

## Navigating (moving a cursor position)

The `Move`* functions provide support for moving a `textpos.Pos` position around the text:
* `MoveForward`, `MoveBackward` and their `*Word` variants move by chars or words.
* `MoveDown` and `MoveUp` take into account the wrapped display lines, and also take a `column` parameter that provides a target column to move along: in editors you may notice that it will try to maintain a target column when moving vertically, even if some of the lines are shorter.


