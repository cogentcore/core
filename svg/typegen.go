// Code generated by "core generate"; DO NOT EDIT.

package svg

import (
	"image"

	"cogentcore.org/core/colors/gradient"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/text/shaped"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
	"github.com/aymerick/douceur/css"
)

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/svg.Circle", IDName: "circle", Doc: "Circle is a SVG circle", Embeds: []types.Field{{Name: "NodeBase"}}, Fields: []types.Field{{Name: "Pos", Doc: "position of the center of the circle"}, {Name: "Radius", Doc: "radius of the circle"}}})

// NewCircle returns a new [Circle] with the given optional parent:
// Circle is a SVG circle
func NewCircle(parent ...tree.Node) *Circle { return tree.New[Circle](parent...) }

// SetPos sets the [Circle.Pos]:
// position of the center of the circle
func (t *Circle) SetPos(v math32.Vector2) *Circle { t.Pos = v; return t }

// SetRadius sets the [Circle.Radius]:
// radius of the circle
func (t *Circle) SetRadius(v float32) *Circle { t.Radius = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/svg.ClipPath", IDName: "clip-path", Doc: "ClipPath is used for holding a path that renders as a clip path", Embeds: []types.Field{{Name: "NodeBase"}}})

// NewClipPath returns a new [ClipPath] with the given optional parent:
// ClipPath is used for holding a path that renders as a clip path
func NewClipPath(parent ...tree.Node) *ClipPath { return tree.New[ClipPath](parent...) }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/svg.StyleSheet", IDName: "style-sheet", Doc: "StyleSheet is a Node2D node that contains a stylesheet -- property values\ncontained in this sheet can be transformed into tree.Properties and set in CSS\nfield of appropriate node", Embeds: []types.Field{{Name: "NodeBase"}}, Fields: []types.Field{{Name: "Sheet"}}})

// NewStyleSheet returns a new [StyleSheet] with the given optional parent:
// StyleSheet is a Node2D node that contains a stylesheet -- property values
// contained in this sheet can be transformed into tree.Properties and set in CSS
// field of appropriate node
func NewStyleSheet(parent ...tree.Node) *StyleSheet { return tree.New[StyleSheet](parent...) }

// SetSheet sets the [StyleSheet.Sheet]
func (t *StyleSheet) SetSheet(v *css.Stylesheet) *StyleSheet { t.Sheet = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/svg.MetaData", IDName: "meta-data", Doc: "MetaData is used for holding meta data info", Embeds: []types.Field{{Name: "NodeBase"}}, Fields: []types.Field{{Name: "MetaData"}}})

// NewMetaData returns a new [MetaData] with the given optional parent:
// MetaData is used for holding meta data info
func NewMetaData(parent ...tree.Node) *MetaData { return tree.New[MetaData](parent...) }

// SetMetaData sets the [MetaData.MetaData]
func (t *MetaData) SetMetaData(v string) *MetaData { t.MetaData = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/svg.Ellipse", IDName: "ellipse", Doc: "Ellipse is a SVG ellipse", Embeds: []types.Field{{Name: "NodeBase"}}, Fields: []types.Field{{Name: "Pos", Doc: "position of the center of the ellipse"}, {Name: "Radii", Doc: "radii of the ellipse in the horizontal, vertical axes"}}})

// NewEllipse returns a new [Ellipse] with the given optional parent:
// Ellipse is a SVG ellipse
func NewEllipse(parent ...tree.Node) *Ellipse { return tree.New[Ellipse](parent...) }

// SetPos sets the [Ellipse.Pos]:
// position of the center of the ellipse
func (t *Ellipse) SetPos(v math32.Vector2) *Ellipse { t.Pos = v; return t }

// SetRadii sets the [Ellipse.Radii]:
// radii of the ellipse in the horizontal, vertical axes
func (t *Ellipse) SetRadii(v math32.Vector2) *Ellipse { t.Radii = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/svg.Filter", IDName: "filter", Doc: "Filter represents SVG filter* elements", Embeds: []types.Field{{Name: "NodeBase"}}, Fields: []types.Field{{Name: "FilterType"}}})

// NewFilter returns a new [Filter] with the given optional parent:
// Filter represents SVG filter* elements
func NewFilter(parent ...tree.Node) *Filter { return tree.New[Filter](parent...) }

// SetFilterType sets the [Filter.FilterType]
func (t *Filter) SetFilterType(v string) *Filter { t.FilterType = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/svg.Flow", IDName: "flow", Doc: "Flow represents SVG flow* elements", Embeds: []types.Field{{Name: "NodeBase"}}, Fields: []types.Field{{Name: "FlowType"}}})

// NewFlow returns a new [Flow] with the given optional parent:
// Flow represents SVG flow* elements
func NewFlow(parent ...tree.Node) *Flow { return tree.New[Flow](parent...) }

// SetFlowType sets the [Flow.FlowType]
func (t *Flow) SetFlowType(v string) *Flow { t.FlowType = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/svg.Gradient", IDName: "gradient", Doc: "Gradient is used for holding a specified color gradient.\nThe name is the id for lookup in url", Embeds: []types.Field{{Name: "NodeBase"}}, Fields: []types.Field{{Name: "Grad", Doc: "the color gradient"}, {Name: "StopsName", Doc: "name of another gradient to get stops from"}}})

// NewGradient returns a new [Gradient] with the given optional parent:
// Gradient is used for holding a specified color gradient.
// The name is the id for lookup in url
func NewGradient(parent ...tree.Node) *Gradient { return tree.New[Gradient](parent...) }

// SetGrad sets the [Gradient.Grad]:
// the color gradient
func (t *Gradient) SetGrad(v gradient.Gradient) *Gradient { t.Grad = v; return t }

// SetStopsName sets the [Gradient.StopsName]:
// name of another gradient to get stops from
func (t *Gradient) SetStopsName(v string) *Gradient { t.StopsName = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/svg.Group", IDName: "group", Doc: "Group groups together SVG elements.\nProvides a common transform for all group elements\nand shared style properties.", Embeds: []types.Field{{Name: "NodeBase"}}})

// NewGroup returns a new [Group] with the given optional parent:
// Group groups together SVG elements.
// Provides a common transform for all group elements
// and shared style properties.
func NewGroup(parent ...tree.Node) *Group { return tree.New[Group](parent...) }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/svg.Image", IDName: "image", Doc: "Image is an SVG image (bitmap)", Embeds: []types.Field{{Name: "NodeBase"}}, Fields: []types.Field{{Name: "Pos", Doc: "position of the top-left of the image"}, {Name: "Size", Doc: "rendered size of the image (imposes a scaling on image when it is rendered)"}, {Name: "Filename", Doc: "file name of image loaded -- set by OpenImage"}, {Name: "ViewBox", Doc: "how to scale and align the image"}, {Name: "Pixels", Doc: "the image pixels"}}})

// NewImage returns a new [Image] with the given optional parent:
// Image is an SVG image (bitmap)
func NewImage(parent ...tree.Node) *Image { return tree.New[Image](parent...) }

// SetPos sets the [Image.Pos]:
// position of the top-left of the image
func (t *Image) SetPos(v math32.Vector2) *Image { t.Pos = v; return t }

// SetSize sets the [Image.Size]:
// rendered size of the image (imposes a scaling on image when it is rendered)
func (t *Image) SetSize(v math32.Vector2) *Image { t.Size = v; return t }

// SetFilename sets the [Image.Filename]:
// file name of image loaded -- set by OpenImage
func (t *Image) SetFilename(v string) *Image { t.Filename = v; return t }

// SetViewBox sets the [Image.ViewBox]:
// how to scale and align the image
func (t *Image) SetViewBox(v ViewBox) *Image { t.ViewBox = v; return t }

// SetPixels sets the [Image.Pixels]:
// the image pixels
func (t *Image) SetPixels(v *image.RGBA) *Image { t.Pixels = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/svg.Line", IDName: "line", Doc: "Line is a SVG line", Embeds: []types.Field{{Name: "NodeBase"}}, Fields: []types.Field{{Name: "Start", Doc: "position of the start of the line"}, {Name: "End", Doc: "position of the end of the line"}}})

// NewLine returns a new [Line] with the given optional parent:
// Line is a SVG line
func NewLine(parent ...tree.Node) *Line { return tree.New[Line](parent...) }

// SetStart sets the [Line.Start]:
// position of the start of the line
func (t *Line) SetStart(v math32.Vector2) *Line { t.Start = v; return t }

// SetEnd sets the [Line.End]:
// position of the end of the line
func (t *Line) SetEnd(v math32.Vector2) *Line { t.End = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/svg.Marker", IDName: "marker", Doc: "Marker represents marker elements that can be drawn along paths (arrow heads, etc)", Embeds: []types.Field{{Name: "NodeBase"}}, Fields: []types.Field{{Name: "RefPos", Doc: "reference position to align the vertex position with, specified in ViewBox coordinates"}, {Name: "Size", Doc: "size of marker to render, in Units units"}, {Name: "Units", Doc: "units to use"}, {Name: "ViewBox", Doc: "viewbox defines the internal coordinate system for the drawing elements within the marker"}, {Name: "Orient", Doc: "orientation of the marker -- either 'auto' or an angle"}, {Name: "VertexPos", Doc: "current vertex position"}, {Name: "VertexAngle", Doc: "current vertex angle in radians"}, {Name: "StrokeWidth", Doc: "current stroke width"}, {Name: "Transform", Doc: "net transform computed from settings and current values -- applied prior to rendering"}, {Name: "EffSize", Doc: "effective size for actual rendering"}}})

// NewMarker returns a new [Marker] with the given optional parent:
// Marker represents marker elements that can be drawn along paths (arrow heads, etc)
func NewMarker(parent ...tree.Node) *Marker { return tree.New[Marker](parent...) }

// SetRefPos sets the [Marker.RefPos]:
// reference position to align the vertex position with, specified in ViewBox coordinates
func (t *Marker) SetRefPos(v math32.Vector2) *Marker { t.RefPos = v; return t }

// SetSize sets the [Marker.Size]:
// size of marker to render, in Units units
func (t *Marker) SetSize(v math32.Vector2) *Marker { t.Size = v; return t }

// SetUnits sets the [Marker.Units]:
// units to use
func (t *Marker) SetUnits(v MarkerUnits) *Marker { t.Units = v; return t }

// SetViewBox sets the [Marker.ViewBox]:
// viewbox defines the internal coordinate system for the drawing elements within the marker
func (t *Marker) SetViewBox(v ViewBox) *Marker { t.ViewBox = v; return t }

// SetOrient sets the [Marker.Orient]:
// orientation of the marker -- either 'auto' or an angle
func (t *Marker) SetOrient(v string) *Marker { t.Orient = v; return t }

// SetVertexPos sets the [Marker.VertexPos]:
// current vertex position
func (t *Marker) SetVertexPos(v math32.Vector2) *Marker { t.VertexPos = v; return t }

// SetVertexAngle sets the [Marker.VertexAngle]:
// current vertex angle in radians
func (t *Marker) SetVertexAngle(v float32) *Marker { t.VertexAngle = v; return t }

// SetStrokeWidth sets the [Marker.StrokeWidth]:
// current stroke width
func (t *Marker) SetStrokeWidth(v float32) *Marker { t.StrokeWidth = v; return t }

// SetTransform sets the [Marker.Transform]:
// net transform computed from settings and current values -- applied prior to rendering
func (t *Marker) SetTransform(v math32.Matrix2) *Marker { t.Transform = v; return t }

// SetEffSize sets the [Marker.EffSize]:
// effective size for actual rendering
func (t *Marker) SetEffSize(v math32.Vector2) *Marker { t.EffSize = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/svg.NodeBase", IDName: "node-base", Doc: "NodeBase is the base type for all elements within an SVG tree.\nIt implements the [Node] interface and contains the core functionality.", Embeds: []types.Field{{Name: "NodeBase"}}, Fields: []types.Field{{Name: "Class", Doc: "Class contains user-defined class name(s) used primarily for attaching\nCSS styles to different display elements.\nMultiple class names can be used to combine properties;\nuse spaces to separate per css standard."}, {Name: "CSS", Doc: "CSS is the cascading style sheet at this level.\nThese styles apply here and to everything below, until superceded.\nUse .class and #name Properties elements to apply entire styles\nto given elements, and type for element type."}, {Name: "CSSAgg", Doc: "CSSAgg is the aggregated css properties from all higher nodes down to this node."}, {Name: "BBox", Doc: "BBox is the bounding box for the node within the SVG Pixels image.\nThis one can be outside the visible range of the SVG image.\nVisBBox is intersected and only shows visible portion."}, {Name: "VisBBox", Doc: "VisBBox is the visible bounding box for the node intersected with the SVG image geometry."}, {Name: "Paint", Doc: "Paint is the paint style information for this node."}, {Name: "isDef", Doc: "isDef is whether this is in [SVG.Defs]."}}})

// NewNodeBase returns a new [NodeBase] with the given optional parent:
// NodeBase is the base type for all elements within an SVG tree.
// It implements the [Node] interface and contains the core functionality.
func NewNodeBase(parent ...tree.Node) *NodeBase { return tree.New[NodeBase](parent...) }

// SetClass sets the [NodeBase.Class]:
// Class contains user-defined class name(s) used primarily for attaching
// CSS styles to different display elements.
// Multiple class names can be used to combine properties;
// use spaces to separate per css standard.
func (t *NodeBase) SetClass(v string) *NodeBase { t.Class = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/svg.Path", IDName: "path", Doc: "Path renders SVG data sequences that can render just about anything", Embeds: []types.Field{{Name: "NodeBase"}}, Fields: []types.Field{{Name: "Data", Doc: "Path data using paint/ppath representation."}, {Name: "DataStr", Doc: "string version of the path data"}}})

// NewPath returns a new [Path] with the given optional parent:
// Path renders SVG data sequences that can render just about anything
func NewPath(parent ...tree.Node) *Path { return tree.New[Path](parent...) }

// SetDataStr sets the [Path.DataStr]:
// string version of the path data
func (t *Path) SetDataStr(v string) *Path { t.DataStr = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/svg.Polygon", IDName: "polygon", Doc: "Polygon is a SVG polygon", Embeds: []types.Field{{Name: "Polyline"}}})

// NewPolygon returns a new [Polygon] with the given optional parent:
// Polygon is a SVG polygon
func NewPolygon(parent ...tree.Node) *Polygon { return tree.New[Polygon](parent...) }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/svg.Polyline", IDName: "polyline", Doc: "Polyline is a SVG multi-line shape", Embeds: []types.Field{{Name: "NodeBase"}}, Fields: []types.Field{{Name: "Points", Doc: "the coordinates to draw -- does a moveto on the first, then lineto for all the rest"}}})

// NewPolyline returns a new [Polyline] with the given optional parent:
// Polyline is a SVG multi-line shape
func NewPolyline(parent ...tree.Node) *Polyline { return tree.New[Polyline](parent...) }

// SetPoints sets the [Polyline.Points]:
// the coordinates to draw -- does a moveto on the first, then lineto for all the rest
func (t *Polyline) SetPoints(v ...math32.Vector2) *Polyline { t.Points = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/svg.Rect", IDName: "rect", Doc: "Rect is a SVG rectangle, optionally with rounded corners", Embeds: []types.Field{{Name: "NodeBase"}}, Fields: []types.Field{{Name: "Pos", Doc: "position of the top-left of the rectangle"}, {Name: "Size", Doc: "size of the rectangle"}, {Name: "Radius", Doc: "radii for curved corners. only rx is used for now."}}})

// NewRect returns a new [Rect] with the given optional parent:
// Rect is a SVG rectangle, optionally with rounded corners
func NewRect(parent ...tree.Node) *Rect { return tree.New[Rect](parent...) }

// SetPos sets the [Rect.Pos]:
// position of the top-left of the rectangle
func (t *Rect) SetPos(v math32.Vector2) *Rect { t.Pos = v; return t }

// SetSize sets the [Rect.Size]:
// size of the rectangle
func (t *Rect) SetSize(v math32.Vector2) *Rect { t.Size = v; return t }

// SetRadius sets the [Rect.Radius]:
// radii for curved corners. only rx is used for now.
func (t *Rect) SetRadius(v math32.Vector2) *Rect { t.Radius = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/svg.Root", IDName: "root", Doc: "Root represents the root of an SVG tree.", Embeds: []types.Field{{Name: "Group"}}, Fields: []types.Field{{Name: "ViewBox", Doc: "ViewBox defines the coordinate system for the drawing.\nThese units are mapped into the screen space allocated\nfor the SVG during rendering."}}})

// NewRoot returns a new [Root] with the given optional parent:
// Root represents the root of an SVG tree.
func NewRoot(parent ...tree.Node) *Root { return tree.New[Root](parent...) }

// SetViewBox sets the [Root.ViewBox]:
// ViewBox defines the coordinate system for the drawing.
// These units are mapped into the screen space allocated
// for the SVG during rendering.
func (t *Root) SetViewBox(v ViewBox) *Root { t.ViewBox = v; return t }

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/svg.Text", IDName: "text", Doc: "Text renders SVG text, handling both text and tspan elements.\ntspan is nested under a parent text, where text has empty Text string.", Embeds: []types.Field{{Name: "NodeBase"}}, Fields: []types.Field{{Name: "Pos", Doc: "position of the left, baseline of the text"}, {Name: "Width", Doc: "width of text to render if using word-wrapping"}, {Name: "Text", Doc: "text string to render"}, {Name: "TextShaped", Doc: "render version of text"}, {Name: "CharPosX", Doc: "character positions along X axis, if specified"}, {Name: "CharPosY", Doc: "character positions along Y axis, if specified"}, {Name: "CharPosDX", Doc: "character delta-positions along X axis, if specified"}, {Name: "CharPosDY", Doc: "character delta-positions along Y axis, if specified"}, {Name: "CharRots", Doc: "character rotations, if specified"}, {Name: "TextLength", Doc: "author's computed text length, if specified -- we attempt to match"}, {Name: "AdjustGlyphs", Doc: "in attempting to match TextLength, should we adjust glyphs in addition to spacing?"}}})

// NewText returns a new [Text] with the given optional parent:
// Text renders SVG text, handling both text and tspan elements.
// tspan is nested under a parent text, where text has empty Text string.
func NewText(parent ...tree.Node) *Text { return tree.New[Text](parent...) }

// SetPos sets the [Text.Pos]:
// position of the left, baseline of the text
func (t *Text) SetPos(v math32.Vector2) *Text { t.Pos = v; return t }

// SetWidth sets the [Text.Width]:
// width of text to render if using word-wrapping
func (t *Text) SetWidth(v float32) *Text { t.Width = v; return t }

// SetText sets the [Text.Text]:
// text string to render
func (t *Text) SetText(v string) *Text { t.Text = v; return t }

// SetTextShaped sets the [Text.TextShaped]:
// render version of text
func (t *Text) SetTextShaped(v *shaped.Lines) *Text { t.TextShaped = v; return t }

// SetCharPosX sets the [Text.CharPosX]:
// character positions along X axis, if specified
func (t *Text) SetCharPosX(v ...float32) *Text { t.CharPosX = v; return t }

// SetCharPosY sets the [Text.CharPosY]:
// character positions along Y axis, if specified
func (t *Text) SetCharPosY(v ...float32) *Text { t.CharPosY = v; return t }

// SetCharPosDX sets the [Text.CharPosDX]:
// character delta-positions along X axis, if specified
func (t *Text) SetCharPosDX(v ...float32) *Text { t.CharPosDX = v; return t }

// SetCharPosDY sets the [Text.CharPosDY]:
// character delta-positions along Y axis, if specified
func (t *Text) SetCharPosDY(v ...float32) *Text { t.CharPosDY = v; return t }

// SetCharRots sets the [Text.CharRots]:
// character rotations, if specified
func (t *Text) SetCharRots(v ...float32) *Text { t.CharRots = v; return t }

// SetTextLength sets the [Text.TextLength]:
// author's computed text length, if specified -- we attempt to match
func (t *Text) SetTextLength(v float32) *Text { t.TextLength = v; return t }

// SetAdjustGlyphs sets the [Text.AdjustGlyphs]:
// in attempting to match TextLength, should we adjust glyphs in addition to spacing?
func (t *Text) SetAdjustGlyphs(v bool) *Text { t.AdjustGlyphs = v; return t }
