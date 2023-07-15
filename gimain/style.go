package gimain

import "github.com/goki/gi/gi"

// StyleFunc is the default style function
// that is called on all widgets to style them.
// It calls the style functions specified in the gi
// packages (gi, giv, svg, etc).
// It is the default value for [gi.MainStyleFunc], so if
// you change [gi.MainStyleFunc], you need to call this function
// at the start of it to keep the default styles
// and build on top of them. If you wish to
// completely remove the default styles, you
// should not call this function in [gi.MainStyleFunc].
func StyleFunc(w *gi.WidgetBase) {
	gi.StyleFunc(w)
}

func init() {
	// StyleFunc is the default value for
	// gi.MainStyleFunc. This has to be done
	// in an init function to avoid
	// an import cycle.
	gi.MainStyleFunc = StyleFunc
}
