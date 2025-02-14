# lines

The lines package manages multi-line monospaced text with a given line width in runes, so that all text wrapping, editing, and navigation logic can be managed purely in text space, allowing rendering and GUI layout to be relatively fast.

This is suitable for text editing and terminal applications, among others. The text encoded as runes along with a corresponding [rich.Text] markup representation with syntax highlighting, using either chroma or the [parse](../parse) package where available. A subsequent update will add support for the [gopls](https://pkg.go.dev/golang.org/x/tools/gopls) system and lsp more generally. The markup is updated in a separate goroutine for efficiency.

Everything is protected by an overall sync.Mutex and is safe to concurrent access, and thus nothing is exported and all access is through protected accessor functions. In general, all unexported methods do NOT lock, and all exported methods do.

## Views

Multiple different views onto the same underlying text content are supported, through the `View` type. Each view can have a different width of characters in its formatting. The `Lines` object manages its own views directly, to ensure everything is updated when the content changes, with a unique ID (int) assigned to each view, which is passed with all view-related methods.

