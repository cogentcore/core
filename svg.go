// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

//go:generate goki generate

import (
	"fmt"
	"image"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"goki.dev/girl/girl"
	"goki.dev/girl/gist"
	"goki.dev/girl/units"
	"goki.dev/ki/v2/ki"
	"goki.dev/mat32/v2"
)

// SVG is a viewport for containing and rendering SVG drawing objects,
// corresponding to the svg tag in html.
// It provides its own Pixels bitmap for drawing into.
type SVG struct {

	// the title of the svg
	Title string `xml:"title" desc:"the title of the svg"`

	// the description of the svg
	Desc string `xml:"desc" desc:"the description of the svg"`

	// fill the viewport with background-color from style
	Fill bool `desc:"fill the viewport with background-color from style"`

	// Size is size of image, Pos is offset within any parent viewport.  Node bounding boxes are based on 0 Pos offset within Pixels image
	Geom girl.Geom2DInt `desc:"Size is size of image, Pos is offset within any parent viewport.  Node bounding boxes are based on 0 Pos offset within Pixels image"`

	// viewbox defines the coordinate system for the drawing -- these units are mapped into the screen space allocated for the SVG during rendering
	ViewBox ViewBox `desc:"viewbox defines the coordinate system for the drawing -- these units are mapped into the screen space allocated for the SVG during rendering"`

	// physical width of the drawing, e.g., when printed -- does not affect rendering -- metadata
	PhysWidth units.Value `desc:"physical width of the drawing, e.g., when printed -- does not affect rendering -- metadata"`

	// physical height of the drawing, e.g., when printed -- does not affect rendering -- metadata
	PhysHeight units.Value `desc:"physical height of the drawing, e.g., when printed -- does not affect rendering -- metadata"`

	// prop: norm = install a transform that renormalizes so that the specified ViewBox exactly fits within the allocated SVG size
	Norm bool `desc:"prop: norm = install a transform that renormalizes so that the specified ViewBox exactly fits within the allocated SVG size"`

	// prop: invert-y = when doing Norm transform, also flip the Y axis so that the smallest Y value is at the bottom of the SVG box, instead of being at the top as it is by default
	InvertY bool `desc:"prop: invert-y = when doing Norm transform, also flip the Y axis so that the smallest Y value is at the bottom of the SVG box, instead of being at the top as it is by default"`

	// [view: -] render state for rendering
	RenderState girl.State `copy:"-" json:"-" xml:"-" view:"-" desc:"render state for rendering"`

	// [view: -] live pixels that we render into
	Pixels *image.RGBA `copy:"-" json:"-" xml:"-" view:"-" desc:"live pixels that we render into"`

	// default paint styles -- inherited by nodes
	Pnt girl.Paint `json:"-" xml:"-" desc:"default paint styles -- inherited by nodes"`

	// all defs defined elements go here (gradients, symbols, etc)
	Defs Group `desc:"all defs defined elements go here (gradients, symbols, etc)"`

	// [view: -] map of def names to index -- uses starting index to find element -- always updated after each search
	DefIdxs map[string]int `view:"-" json:"-" xml:"-" desc:"map of def names to index -- uses starting index to find element -- always updated after each search"`

	// [view: -] map of unique numeric ids for all elements -- used for allocating new unique id numbers, appended to end of elements -- see NewUniqueId, GatherIds
	UniqueIds map[int]struct{} `view:"-" json:"-" xml:"-" desc:"map of unique numeric ids for all elements -- used for allocating new unique id numbers, appended to end of elements -- see NewUniqueId, GatherIds"`

	// [view: -] mutex for protecting rendering
	RenderMu sync.Mutex `view:"-" json:"-" xml:"-" desc:"mutex for protecting rendering"`
}

// todo:
// func (sv *SVG) OnInit() {
// 	sv.AddStyler(func(w *gi.WidgetBase, s *gist.Style) {
// 		if par := sv.ParentWidget(); par != nil {
// 			sv.Pnt.FillStyle.Color.SetColor(par.Style.Color)
// 			sv.Pnt.StrokeStyle.Color.SetColor(par.Style.Color)
// 		}
// 	})
// }

// NewSVG creates a SVG with Pixels Image of the specified width and height
func NewSVG(width, height int) *SVG {
	sz := image.Point{width, height}
	vp := &SVG{
		Geom: Geom2DInt{Size: sz},
	}
	vp.Pixels = image.NewRGBA(image.Rectangle{Max: sz})
	vp.Render.Init(width, height, vp.Pixels)
	return vp
}

// Resize resizes the viewport, creating a new image -- updates Geom Size
func (sv *SVG) Resize(nwsz image.Point) {
	if nwsz.X == 0 || nwsz.Y == 0 {
		return
	}
	if sv.Pixels != nil {
		ib := sv.Pixels.Bounds().Size()
		if ib == nwsz {
			sv.Geom.Size = nwsz // make sure
			return              // already good
		}
	}
	if sv.Pixels != nil {
		sv.Pixels = nil
	}
	sv.Pixels = image.NewRGBA(image.Rectangle{Max: nwsz})
	sv.RenderState.Init(nwsz.X, nwsz.Y, sv.Pixels)
	sv.Geom.Size = nwsz // make sure
}

func (sv *SVG) CopyFieldsFrom(frm any) {
	fr := frm.(*SVG)
	sv.Title = fr.Title
	sv.Desc = fr.Desc
	sv.Fill = fr.Fill
	sv.Geom = fr.Geom
	sv.ViewBox = fr.ViewBox
	sv.Norm = fr.Norm
	sv.InvertY = fr.InvertY
	sv.Pnt = fr.Pnt
	sv.Defs.CopyFrom(&fr.Defs)
	sv.UniqueIds = nil
}

// // Paint satisfies the painter interface
// func (sv *SVG) Paint() *gist.Paint {
// 	return &sv.Pnt.Paint
// }

// DeleteAll deletes any existing elements in this svg
func (sv *SVG) DeleteAll() {
	updt := sv.UpdateStart()
	sv.DeleteChildren(ki.DestroyKids)
	sv.ViewBox.Defaults()
	sv.Pnt.Defaults()
	sv.Defs.DeleteChildren(ki.DestroyKids)
	sv.Title = ""
	sv.Desc = ""
	sv.UpdateEnd(updt)
}

// SetNormXForm sets a scaling transform to make the entire viewbox to fit the viewport
func (sv *SVG) SetNormXForm() {
	pc := &sv.Pnt
	pc.XForm = mat32.Identity2D()
	if sv.ViewBox.Size != mat32.Vec2Zero {
		// todo: deal with all the other options!
		vpsX := float32(sv.Geom.Size.X) / sv.ViewBox.Size.X
		vpsY := float32(sv.Geom.Size.Y) / sv.ViewBox.Size.Y
		if sv.InvertY {
			vpsY *= -1
		}
		sv.Pnt.XForm = sv.Pnt.XForm.Scale(vpsX, vpsY).Translate(-sv.ViewBox.Min.X, -sv.ViewBox.Min.Y)
		if sv.InvertY {
			sv.Pnt.XForm.Y0 = -sv.Pnt.XForm.Y0
		}
	}
}

// SetDPIXForm sets a scaling transform to compensate for the dpi -- svg
// rendering is done within a 96 DPI context
func (sv *SVG) SetDPIXForm() {
	pc := &sv.Pnt
	dpisc := sv.ParentWindow().LogicalDPI() / 96.0
	pc.XForm = mat32.Scale2D(dpisc, dpisc)
}

func (sv *SVG) Init2D() {
	sv.Viewport2D.Init2D()
	sv.Pnt.Defaults()
	// sv.Pnt.FontStyle.BackgroundColor.SetSolid(gist.White)
}

// SetUnitContext sets the unit context based on size of viewport, element,
// and parent element (from bbox) and then caches everything out in terms of raw pixel
// dots for rendering -- call at start of render
func SetUnitContext(pc *gist.Paint, sv *SVG, el, par mat32.Vec2) {
	pc.UnContext.Defaults()
	if vp != nil {
		pc.UnContext.DPI = 96 // paint (SVG) context is always 96 = 1to1
		if vp.Render.Image != nil {
			sz := vp.Render.Image.Bounds().Size()
			pc.UnContext.SetSizes(float32(sz.X), float32(sz.Y), el.X, el.Y, par.X, par.Y)
		} else {
			pc.UnContext.SetSizes(0, 0, el.X, el.Y, par.X, par.Y)
		}
	}
	pc.FontStyle.SetUnitContext(&pc.UnContext)
	pc.ToDots()
}

// ContextColorSpecByURL finds a Node by an element name (URL-like path), and
// attempts to convert it to a Gradient -- if successful, returns ColorSpec on that.
// Used for colorspec styling based on url() value.
func (sv *SVG) ContextColorSpecByURL(url string) *gist.ColorSpec {
	if sv == nil {
		return nil
	}
	val := url[4:]
	val = strings.TrimPrefix(strings.TrimSuffix(val, ")"), "#")
	def := sv.FindDefByName(val)
	if def != nil {
		if grad, ok := def.(*Gradient); ok {
			return &grad.Grad
		}
	}
	if sv.CurStyleNode == nil {
		return nil
	}
	ne := sv.CurStyleNode.FindNamedElement(val)
	if grad, ok := ne.(*Gradient); ok {
		return &grad.Grad
	}
	return nil
}

func (sv *SVG) Style() {
	sv.Pnt.Defaults()
	// TODO: cleaner svg styling from text color property
	SetUnitContext(&sv.Pnt.Paint, sv, mat32.Vec2{}, mat32.Vec2{})
	// STYTODO: maybe pass something in here using viewbox as context?

	sv.FuncDownMeFirst(0, nb.This(), func(k ki.Ki, level int, d any) bool {
		if k == sv.This() {
			return ki.Continue
		}
		ni := k.(Node)
		if ni == nil || ni.IsDeleted() || ni.IsDestroyed() {
			return ki.Break
		}
		ni.Style(sv)
		return ki.Continue
	})

}

// func (sv *SVG) Style2D() {
// 	if nv, err := sv.PropTry("norm"); err == nil {
// 		sv.Norm, _ = kit.ToBool(nv)
// 	}
// 	if iv, err := sv.PropTry("invert-y"); err == nil {
// 		sv.InvertY, _ = kit.ToBool(iv)
// 	}
// }

func (sv *SVG) Render() {
	sv.RenderMu.Lock()
	defer sv.RenderMu.Unlock()

	sv.Style()

	rs := &sv.RenderState
	if sv.Fill {
		sv.FillViewport()
	}
	if sv.Norm {
		sv.SetNormXForm()
	}
	rs.PushXForm(sv.Pnt.XForm)

	for _, kid := range sv.Kids {
		ni := kid.(Node)
		ni.Render()
	}

	rs.PopXForm()
}

/////////////////////////////////////////////////////////////////////////////
//   Naming elements with unique id's

// SplitNameIdDig splits name into numerical end part and preceding name,
// based on string of digits from end of name.
// If Id == 0 then it was not specified or didn't parse.
// SVG object names are element names + numerical id
func SplitNameIdDig(nm string) (string, int) {
	sz := len(nm)

	for i := sz - 1; i >= 0; i-- {
		c := rune(nm[i])
		if !unicode.IsDigit(c) {
			if i == sz-1 {
				return nm, 0
			}
			n := nm[:i+1]
			id, _ := strconv.Atoi(nm[i+1:])
			return n, id
		}
	}
	return nm, 0
}

// SplitNameId splits name after the element name (e.g., 'rect')
// returning true if it starts with element name,
// and numerical id part after that element.
// if numerical id part is 0, then it didn't parse.
// SVG object names are element names + numerical id
func SplitNameId(elnm, nm string) (bool, int) {
	if !strings.HasPrefix(nm, elnm) {
		// fmt.Printf("not elnm: %s  %s\n", nm, elnm)
		return false, 0
	}
	idstr := nm[len(elnm):]
	id, _ := strconv.Atoi(idstr)
	return true, id
}

// NameId returns the name with given unique id.
// returns plain name if id == 0
func NameId(nm string, id int) string {
	if id == 0 {
		return nm
	}
	return fmt.Sprintf("%s%d", nm, id)
}

// GatherIds gathers all the numeric id suffixes currently in use.
// It automatically renames any that are not unique or empty.
func (sv *SVG) GatherIds() {
	sv.UniqueIds = make(map[int]struct{})
	sv.FuncDownMeFirst(0, nil, func(k ki.Ki, level int, d any) bool {
		sv.NodeEnsureUniqueId(k)
		return ki.Continue
	})
}

// NodeEnsureUniqueId ensures that the given node has a unique Id
// Call this on any newly-created nodes.
func (sv *SVG) NodeEnsureUniqueId(kn ki.Ki) {
	elnm := ""
	svi, issvi := kn.(Node)
	if issvi {
		elnm = svi.SVGName()
		// } else if gr, ok := kn.(*Gradient); ok { // don't need gradients to be unique
		// 	elnm = gr.GradientType()
	}
	if elnm == "" {
		return
	}
	elpfx, id := SplitNameId(elnm, kn.Name())
	if !elpfx {
		if issvi {
			if !svi.EnforceSVGName() { // if we end in a number, just register it anyway
				_, id = SplitNameIdDig(kn.Name())
				if id > 0 {
					sv.UniqueIds[id] = struct{}{}
				}
				return
			}
		}
		_, id = SplitNameIdDig(kn.Name())
		if id > 0 {
			kn.SetName(NameId(elnm, id))
			kn.UpdateSig()
		}
	}
	_, exists := sv.UniqueIds[id]
	if id <= 0 || exists {
		id = sv.NewUniqueId() // automatically registers it
		kn.SetName(NameId(elnm, id))
		kn.UpdateSig()
	} else {
		sv.UniqueIds[id] = struct{}{}
	}
}

// NewUniqueId returns a new unique numerical id number, for naming an object
func (sv *SVG) NewUniqueId() int {
	if sv.UniqueIds == nil {
		sv.GatherIds()
	}
	sz := len(sv.UniqueIds)
	var nid int
	for {
		switch {
		case sz >= 10000:
			nid = rand.Intn(sz * 100)
		case sz >= 1000:
			nid = rand.Intn(10000)
		default:
			nid = rand.Intn(1000)
		}
		if _, has := sv.UniqueIds[nid]; has {
			continue
		}
		break
	}
	sv.UniqueIds[nid] = struct{}{}
	return nid
}

/*
// SVGFlags extend gi.VpFlags to hold SVG node state
type SVGFlags int

var TypeSVGFlags = kit.Enums.AddEnumExt(gi.TypeVpFlags, SVGFlagsN, kit.BitFlag, nil)

const (
	// Rendering means that the SVG is currently redrawing
	// Can be useful to check for animations etc to decide whether to
	// drive another update
	Rendering SVGFlags = SVGFlags(gi.VpFlagsN) + iota

	SVGFlagsN
)

var SVGProps = ki.Props{
	ki.EnumTypeFlag: TypeSVGFlags,
	"ToolBar": ki.PropSlice{
		{"OpenXML", ki.Props{
			"label": "Open...",
			"desc":  "Open SVG XML-formatted file",
			"icon":  icons.FileOpen,
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"ext": ".svg",
				}},
			},
		}},
		{"SaveXML", ki.Props{
			"label": "SaveAs...",
			"desc":  "Save SVG content to an XML-formatted file.",
			"icon":  icons.SaveAs,
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"ext": ".svg",
				}},
			},
		}},
	},
}
*/
