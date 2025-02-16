// Code generated by "core generate"; DO NOT EDIT.

package textcore

import (
	"image"

	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/text/textcore.Base", IDName: "base", Doc: "Base is a widget with basic infrastructure for viewing and editing\n[lines.Lines] of monospaced text, used in [texteditor.Editor] and\nterminal. There can be multiple Base widgets for each lines buffer.\n\nUse NeedsRender to drive an render update for any change that does\nnot change the line-level layout of the text.\n\nAll updating in the Base should be within a single goroutine,\nas it would require extensive protections throughout code otherwise.", Directives: []types.Directive{{Tool: "core", Directive: "embedder"}}, Embeds: []types.Field{{Name: "Frame"}}, Fields: []types.Field{{Name: "Lines", Doc: "Lines is the text lines content for this editor."}, {Name: "CursorWidth", Doc: "CursorWidth is the width of the cursor.\nThis should be set in Stylers like all other style properties."}, {Name: "LineNumberColor", Doc: "LineNumberColor is the color used for the side bar containing the line numbers.\nThis should be set in Stylers like all other style properties."}, {Name: "SelectColor", Doc: "SelectColor is the color used for the user text selection background color.\nThis should be set in Stylers like all other style properties."}, {Name: "HighlightColor", Doc: "HighlightColor is the color used for the text highlight background color (like in find).\nThis should be set in Stylers like all other style properties."}, {Name: "CursorColor", Doc: "CursorColor is the color used for the text editor cursor bar.\nThis should be set in Stylers like all other style properties."}, {Name: "renders", Doc: "renders is a slice of shaped.Lines representing the renders of the\nvisible text lines, with one render per line (each line could visibly\nwrap-around, so these are logical lines, not display lines)."}, {Name: "viewId", Doc: "viewId is the unique id of the Lines view."}, {Name: "charSize", Doc: "charSize is the render size of one character (rune).\nY = line height, X = total glyph advance."}, {Name: "visSizeAlloc", Doc: "visSizeAlloc is the Geom.Size.Alloc.Total subtracting extra space,\navailable for rendering text lines and line numbers."}, {Name: "lastVisSizeAlloc", Doc: "lastVisSizeAlloc is the last visSizeAlloc used in laying out lines.\nIt is used to trigger a new layout only when needed."}, {Name: "visSize", Doc: "visSize is the height in lines and width in chars of the visible area."}, {Name: "linesSize", Doc: "linesSize is the height in lines and width in chars of the Lines text area,\n(excluding line numbers), which can be larger than the visSize."}, {Name: "scrollPos", Doc: "scrollPos is the position of the scrollbar, in units of lines of text.\nfractional scrolling is supported."}, {Name: "lineNumberOffset", Doc: "lineNumberOffset is the horizontal offset in chars for the start of text\nafter line numbers. This is 0 if no line numbers."}, {Name: "totalSize", Doc: "totalSize is total size of all text, including line numbers,\nmultiplied by charSize."}, {Name: "lineNumberDigits", Doc: "lineNumberDigits is the number of line number digits needed."}, {Name: "lineNumberRenders", Doc: "lineNumberRenders are the renderers for line numbers, per visible line."}, {Name: "CursorPos", Doc: "CursorPos is the current cursor position."}, {Name: "lastFilename", Doc: "\t\t// cursorTarget is the target cursor position for externally set targets.\n\t\t// It ensures that the target position is visible.\n\t\tcursorTarget textpos.Pos\n\n\t\t// cursorColumn is the desired cursor column, where the cursor was last when moved using left / right arrows.\n\t\t// It is used when doing up / down to not always go to short line columns.\n\t\tcursorColumn int\n\n\t\t// posHistoryIndex is the current index within PosHistory.\n\t\tposHistoryIndex int\n\n\t\t// selectStart is the starting point for selection, which will either be the start or end of selected region\n\t\t// depending on subsequent selection.\n\t\tselectStart textpos.Pos\n\n\t\t// SelectRegion is the current selection region.\n\t\tSelectRegion lines.Region `set:\"-\" edit:\"-\" json:\"-\" xml:\"-\"`\n\n\t\t// previousSelectRegion is the previous selection region that was actually rendered.\n\t\t// It is needed to update the render.\n\t\tpreviousSelectRegion lines.Region\n\n\t\t// Highlights is a slice of regions representing the highlighted regions, e.g., for search results.\n\t\tHighlights []lines.Region `set:\"-\" edit:\"-\" json:\"-\" xml:\"-\"`\n\n\t\t// scopelights is a slice of regions representing the highlighted regions specific to scope markers.\n\t\tscopelights []lines.Region\n\n\t\t// LinkHandler handles link clicks.\n\t\t// If it is nil, they are sent to the standard web URL handler.\n\t\tLinkHandler func(tl *rich.Link)\n\n\t\t// ISearch is the interactive search data.\n\t\tISearch ISearch `set:\"-\" edit:\"-\" json:\"-\" xml:\"-\"`\n\n\t\t// QReplace is the query replace data.\n\t\tQReplace QReplace `set:\"-\" edit:\"-\" json:\"-\" xml:\"-\"`\n\n\t\t// selectMode is a boolean indicating whether to select text as the cursor moves.\n\t\tselectMode bool\n\n\t\t// blinkOn oscillates between on and off for blinking.\n\t\tblinkOn bool\n\n\t\t// cursorMu is a mutex protecting cursor rendering, shared between blink and main code.\n\t\tcursorMu sync.Mutex\n\n\t\t// hasLinks is a boolean indicating if at least one of the renders has links.\n\t\t// It determines if we set the cursor for hand movements.\n\t\thasLinks bool\n\n\t\t// hasLineNumbers indicates that this editor has line numbers\n\t\t// (per [Buffer] option)\n\t\thasLineNumbers bool // TODO: is this really necessary?\n\n\t\t// lastWasTabAI indicates that last key was a Tab auto-indent\n\t\tlastWasTabAI bool\n\n\t\t// lastWasUndo indicates that last key was an undo\n\t\tlastWasUndo bool\n\n\t\t// targetSet indicates that the CursorTarget is set\n\t\ttargetSet bool\n\n\t\tlastRecenter   int\n\t\tlastAutoInsert rune"}}})

// NewBase returns a new [Base] with the given optional parent:
// Base is a widget with basic infrastructure for viewing and editing
// [lines.Lines] of monospaced text, used in [texteditor.Editor] and
// terminal. There can be multiple Base widgets for each lines buffer.
//
// Use NeedsRender to drive an render update for any change that does
// not change the line-level layout of the text.
//
// All updating in the Base should be within a single goroutine,
// as it would require extensive protections throughout code otherwise.
func NewBase(parent ...tree.Node) *Base { return tree.New[Base](parent...) }

// BaseEmbedder is an interface that all types that embed Base satisfy
type BaseEmbedder interface {
	AsBase() *Base
}

// AsBase returns the given value as a value of type Base if the type
// of the given value embeds Base, or nil otherwise
func AsBase(n tree.Node) *Base {
	if t, ok := n.(BaseEmbedder); ok {
		return t.AsBase()
	}
	return nil
}

// AsBase satisfies the [BaseEmbedder] interface
func (t *Base) AsBase() *Base { return t }

// SetCursorWidth sets the [Base.CursorWidth]:
// CursorWidth is the width of the cursor.
// This should be set in Stylers like all other style properties.
func (t *Base) SetCursorWidth(v units.Value) *Base { t.CursorWidth = v; return t }

// SetLineNumberColor sets the [Base.LineNumberColor]:
// LineNumberColor is the color used for the side bar containing the line numbers.
// This should be set in Stylers like all other style properties.
func (t *Base) SetLineNumberColor(v image.Image) *Base { t.LineNumberColor = v; return t }

// SetSelectColor sets the [Base.SelectColor]:
// SelectColor is the color used for the user text selection background color.
// This should be set in Stylers like all other style properties.
func (t *Base) SetSelectColor(v image.Image) *Base { t.SelectColor = v; return t }

// SetHighlightColor sets the [Base.HighlightColor]:
// HighlightColor is the color used for the text highlight background color (like in find).
// This should be set in Stylers like all other style properties.
func (t *Base) SetHighlightColor(v image.Image) *Base { t.HighlightColor = v; return t }

// SetCursorColor sets the [Base.CursorColor]:
// CursorColor is the color used for the text editor cursor bar.
// This should be set in Stylers like all other style properties.
func (t *Base) SetCursorColor(v image.Image) *Base { t.CursorColor = v; return t }
