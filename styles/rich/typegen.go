// Code generated by "core generate -add-types -setters"; DO NOT EDIT.

package rich

import (
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/types"
)

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/styles/rich.Context", IDName: "context", Doc: "Context holds the global context for rich text styling,\nholding properties that apply to a collection of [rich.Text] elements,\nso it does not need to be redundantly encoded in each such element.", Fields: []types.Field{{Name: "Standard", Doc: "Standard is the standard font size. The Style provides a multiplier\non this value."}, {Name: "SansSerif", Doc: "SansSerif is a font without serifs, where glyphs have plain stroke endings,\nwithout ornamentation. Example sans-serif fonts include Arial, Helvetica,\nOpen Sans, Fira Sans, Lucida Sans, Lucida Sans Unicode, Trebuchet MS,\nLiberation Sans, and Nimbus Sans L.\nThis can be a list of comma-separated names, tried in order.\n\"sans-serif\" will be added automatically as a final backup."}, {Name: "Serif", Doc: "Serif is a small line or stroke attached to the end of a larger stroke\nin a letter. In serif fonts, glyphs have finishing strokes, flared or\ntapering ends. Examples include Times New Roman, Lucida Bright,\nLucida Fax, Palatino, Palatino Linotype, Palladio, and URW Palladio.\nThis can be a list of comma-separated names, tried in order.\n\"serif\" will be added automatically as a final backup."}, {Name: "Monospace", Doc: "Monospace fonts have all glyphs with he same fixed width.\nExample monospace fonts include Fira Mono, DejaVu Sans Mono,\nMenlo, Consolas, Liberation Mono, Monaco, and Lucida Console.\nThis can be a list of comma-separated names. serif will be added\nautomatically as a final backup.\nThis can be a list of comma-separated names, tried in order.\n\"monospace\" will be added automatically as a final backup."}, {Name: "Cursive", Doc: "Cursive glyphs generally have either joining strokes or other cursive\ncharacteristics beyond those of italic typefaces. The glyphs are partially\nor completely connected, and the result looks more like handwritten pen or\nbrush writing than printed letter work. Example cursive fonts include\nBrush Script MT, Brush Script Std, Lucida Calligraphy, Lucida Handwriting,\nand Apple Chancery.\nThis can be a list of comma-separated names, tried in order.\n\"cursive\" will be added automatically as a final backup."}, {Name: "Fantasy", Doc: "Fantasy fonts are primarily decorative fonts that contain playful\nrepresentations of characters. Example fantasy fonts include Papyrus,\nHerculanum, Party LET, Curlz MT, and Harrington.\nThis can be a list of comma-separated names, tried in order.\n\"fantasy\" will be added automatically as a final backup."}, {Name: "Math", Doc: "\tMath fonts are for displaying mathematical expressions, for example\nsuperscript and subscript, brackets that cross several lines, nesting\nexpressions, and double-struck glyphs with distinct meanings.\nThis can be a list of comma-separated names, tried in order.\n\"math\" will be added automatically as a final backup."}, {Name: "Emoji", Doc: "Emoji fonts are specifically designed to render emoji.\nThis can be a list of comma-separated names, tried in order.\n\"emoji\" will be added automatically as a final backup."}, {Name: "Fangsong", Doc: "Fangsong are a particular style of Chinese characters that are between\nserif-style Song and cursive-style Kai forms. This style is often used\nfor government documents.\nThis can be a list of comma-separated names, tried in order.\n\"fangsong\" will be added automatically as a final backup."}, {Name: "Custom", Doc: "Custom is a custom font name."}}})

// SetStandard sets the [Context.Standard]:
// Standard is the standard font size. The Style provides a multiplier
// on this value.
func (t *Context) SetStandard(v units.Value) *Context { t.Standard = v; return t }

// SetSansSerif sets the [Context.SansSerif]:
// SansSerif is a font without serifs, where glyphs have plain stroke endings,
// without ornamentation. Example sans-serif fonts include Arial, Helvetica,
// Open Sans, Fira Sans, Lucida Sans, Lucida Sans Unicode, Trebuchet MS,
// Liberation Sans, and Nimbus Sans L.
// This can be a list of comma-separated names, tried in order.
// "sans-serif" will be added automatically as a final backup.
func (t *Context) SetSansSerif(v string) *Context { t.SansSerif = v; return t }

// SetSerif sets the [Context.Serif]:
// Serif is a small line or stroke attached to the end of a larger stroke
// in a letter. In serif fonts, glyphs have finishing strokes, flared or
// tapering ends. Examples include Times New Roman, Lucida Bright,
// Lucida Fax, Palatino, Palatino Linotype, Palladio, and URW Palladio.
// This can be a list of comma-separated names, tried in order.
// "serif" will be added automatically as a final backup.
func (t *Context) SetSerif(v string) *Context { t.Serif = v; return t }

// SetMonospace sets the [Context.Monospace]:
// Monospace fonts have all glyphs with he same fixed width.
// Example monospace fonts include Fira Mono, DejaVu Sans Mono,
// Menlo, Consolas, Liberation Mono, Monaco, and Lucida Console.
// This can be a list of comma-separated names. serif will be added
// automatically as a final backup.
// This can be a list of comma-separated names, tried in order.
// "monospace" will be added automatically as a final backup.
func (t *Context) SetMonospace(v string) *Context { t.Monospace = v; return t }

// SetCursive sets the [Context.Cursive]:
// Cursive glyphs generally have either joining strokes or other cursive
// characteristics beyond those of italic typefaces. The glyphs are partially
// or completely connected, and the result looks more like handwritten pen or
// brush writing than printed letter work. Example cursive fonts include
// Brush Script MT, Brush Script Std, Lucida Calligraphy, Lucida Handwriting,
// and Apple Chancery.
// This can be a list of comma-separated names, tried in order.
// "cursive" will be added automatically as a final backup.
func (t *Context) SetCursive(v string) *Context { t.Cursive = v; return t }

// SetFantasy sets the [Context.Fantasy]:
// Fantasy fonts are primarily decorative fonts that contain playful
// representations of characters. Example fantasy fonts include Papyrus,
// Herculanum, Party LET, Curlz MT, and Harrington.
// This can be a list of comma-separated names, tried in order.
// "fantasy" will be added automatically as a final backup.
func (t *Context) SetFantasy(v string) *Context { t.Fantasy = v; return t }

// SetMath sets the [Context.Math]:
//
//	Math fonts are for displaying mathematical expressions, for example
//
// superscript and subscript, brackets that cross several lines, nesting
// expressions, and double-struck glyphs with distinct meanings.
// This can be a list of comma-separated names, tried in order.
// "math" will be added automatically as a final backup.
func (t *Context) SetMath(v string) *Context { t.Math = v; return t }

// SetEmoji sets the [Context.Emoji]:
// Emoji fonts are specifically designed to render emoji.
// This can be a list of comma-separated names, tried in order.
// "emoji" will be added automatically as a final backup.
func (t *Context) SetEmoji(v string) *Context { t.Emoji = v; return t }

// SetFangsong sets the [Context.Fangsong]:
// Fangsong are a particular style of Chinese characters that are between
// serif-style Song and cursive-style Kai forms. This style is often used
// for government documents.
// This can be a list of comma-separated names, tried in order.
// "fangsong" will be added automatically as a final backup.
func (t *Context) SetFangsong(v string) *Context { t.Fangsong = v; return t }

// SetCustom sets the [Context.Custom]:
// Custom is a custom font name.
func (t *Context) SetCustom(v string) *Context { t.Custom = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/styles/rich.Style", IDName: "style", Doc: "Style contains all of the rich text styling properties, that apply to one\nspan of text. These are encoded into a uint32 rune value in [rich.Text].\nSee [Context] for additional context needed for full specification.", Directives: []types.Directive{{Tool: "go", Directive: "generate", Args: []string{"core", "generate", "-add-types", "-setters"}}, {Tool: "types", Directive: "add"}}, Fields: []types.Field{{Name: "Size", Doc: "Size is the font size multiplier relative to the standard font size\nspecified in the Context."}, {Name: "Family", Doc: "Family indicates the generic family of typeface to use, where the\nspecific named values to use for each are provided in the Context."}, {Name: "Slant", Doc: "Slant allows italic or oblique faces to be selected."}, {Name: "Weight", Doc: "Weights are the degree of blackness or stroke thickness of a font.\nThis value ranges from 100.0 to 900.0, with 400.0 as normal."}, {Name: "Stretch", Doc: "Stretch is the width of a font as an approximate fraction of the normal width.\nWidths range from 0.5 to 2.0 inclusive, with 1.0 as the normal width."}, {Name: "Special", Doc: "Special additional formatting factors that are not otherwise\ncaptured by changes in font rendering properties or decorations."}, {Name: "Decoration", Doc: "Decorations are underline, line-through, etc, as bit flags\nthat must be set using [Decorations.SetFlag]."}, {Name: "FillColor", Doc: "\tFillColor is the color to use for glyph fill (i.e., the standard \"ink\" color)\nif the Decoration FillColor flag is set. This will be encoded in a uint32 following\nthe style rune, in rich.Text spans."}, {Name: "StrokeColor", Doc: "\tStrokeColor is the color to use for glyph stroking if the Decoration StrokeColor\nflag is set. This will be encoded in a uint32 following the style rune,\nin rich.Text spans."}, {Name: "Background", Doc: "\tBackground is the color to use for the background region if the Decoration\nBackground flag is set. This will be encoded in a uint32 following the style rune,\nin rich.Text spans."}}})

// SetSize sets the [Style.Size]:
// Size is the font size multiplier relative to the standard font size
// specified in the Context.
func (t *Style) SetSize(v float32) *Style { t.Size = v; return t }

// SetFamily sets the [Style.Family]:
// Family indicates the generic family of typeface to use, where the
// specific named values to use for each are provided in the Context.
func (t *Style) SetFamily(v Family) *Style { t.Family = v; return t }

// SetSlant sets the [Style.Slant]:
// Slant allows italic or oblique faces to be selected.
func (t *Style) SetSlant(v Slants) *Style { t.Slant = v; return t }

// SetWeight sets the [Style.Weight]:
// Weights are the degree of blackness or stroke thickness of a font.
// This value ranges from 100.0 to 900.0, with 400.0 as normal.
func (t *Style) SetWeight(v Weights) *Style { t.Weight = v; return t }

// SetStretch sets the [Style.Stretch]:
// Stretch is the width of a font as an approximate fraction of the normal width.
// Widths range from 0.5 to 2.0 inclusive, with 1.0 as the normal width.
func (t *Style) SetStretch(v Stretch) *Style { t.Stretch = v; return t }

// SetSpecial sets the [Style.Special]:
// Special additional formatting factors that are not otherwise
// captured by changes in font rendering properties or decorations.
func (t *Style) SetSpecial(v Specials) *Style { t.Special = v; return t }

// SetDecoration sets the [Style.Decoration]:
// Decorations are underline, line-through, etc, as bit flags
// that must be set using [Decorations.SetFlag].
func (t *Style) SetDecoration(v Decorations) *Style { t.Decoration = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/styles/rich.Family", IDName: "family", Doc: "Family specifies the generic family of typeface to use, where the\nspecific named values to use for each are provided in the Context."})

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/styles/rich.Slants", IDName: "slants", Doc: "Slants (also called style) allows italic or oblique faces to be selected."})

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/styles/rich.Weights", IDName: "weights", Doc: "Weights are the degree of blackness or stroke thickness of a font.\nThis value ranges from 100.0 to 900.0, with 400.0 as normal."})

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/styles/rich.Stretch", IDName: "stretch", Doc: "Stretch is the width of a font as an approximate fraction of the normal width.\nWidths range from 0.5 to 2.0 inclusive, with 1.0 as the normal width."})

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/styles/rich.Decorations", IDName: "decorations", Doc: "Decorations are underline, line-through, etc, as bit flags\nthat must be set using [Font.SetDecoration]."})

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/styles/rich.Specials", IDName: "specials", Doc: "Specials are special additional formatting factors that are not\notherwise captured by changes in font rendering properties or decorations."})

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/styles/rich.Text", IDName: "text", Doc: "Text is a rich text representation, with spans of []rune unicode characters\nthat share a common set of text styling properties, which are represented\nby the first rune(s) in each span. If custom colors are used, they are encoded\nafter the first style rune.\nThis compact and efficient representation can be Join'd back into the raw\nunicode source, and indexing by rune index in the original is fast.\nIt provides a GPU-compatible representation, and is the text equivalent of\nthe [ppath.Path] encoding."})

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/styles/rich.Index", IDName: "index", Doc: "Index represents the [Span][Rune] index of a given rune.\nThe Rune index can be either the actual index for [Text], taking\ninto account the leading style rune(s), or the logical index\ninto a [][]rune type with no style runes, depending on the context.", Directives: []types.Directive{{Tool: "types", Directive: "add"}}, Fields: []types.Field{{Name: "Span"}, {Name: "Rune"}}})

// SetSpan sets the [Index.Span]
func (t *Index) SetSpan(v int) *Index { t.Span = v; return t }

// SetRune sets the [Index.Rune]
func (t *Index) SetRune(v int) *Index { t.Rune = v; return t }
