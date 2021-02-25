// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"fmt"
	"image"
	"math/rand"
	"strconv"
	"strings"
	"unicode"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/girl"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// see io.go for IO input / output methods

// SVG is a viewport for containing SVG drawing objects, corresponding to the
// svg tag in html -- it provides its own bitmap for drawing into.
// To trigger a full re-render of SVG, do SetNeedsFullRender()
// in UpdateStart / End loop.
type SVG struct {
	gi.Viewport2D
	ViewBox    ViewBox          `desc:"viewbox defines the coordinate system for the drawing -- these units are mapped into the screen space allocated for the SVG during rendering"`
	PhysWidth  units.Value      `desc:"physical width of the drawing, e.g., when printed -- does not affect rendering -- metadata"`
	PhysHeight units.Value      `desc:"physical height of the drawing, e.g., when printed -- does not affect rendering -- metadata"`
	Norm       bool             `desc:"prop: norm = install a transform that renormalizes so that the specified ViewBox exactly fits within the allocated SVG size"`
	InvertY    bool             `desc:"prop: invert-y = when doing Norm transform, also flip the Y axis so that the smallest Y value is at the bottom of the SVG box, instead of being at the top as it is by default"`
	Pnt        girl.Paint       `json:"-" xml:"-" desc:"paint styles -- inherited by nodes"`
	Defs       Group            `desc:"all defs defined elements go here (gradients, symbols, etc)"`
	Title      string           `xml:"title" desc:"the title of the svg"`
	Desc       string           `xml:"desc" desc:"the description of the svg"`
	DefIdxs    map[string]int   `view:"-" json:"-" xml:"-" desc:"map of def names to index -- uses starting index to find element -- always updated after each search"`
	UniqueIds  map[int]struct{} `view:"-" json:"-" xml:"-" desc:"map of unique numeric ids for all elements -- used for allocating new unique id numbers, appended to end of elements -- see NewUniqueId, GatherIds"`
}

var KiT_SVG = kit.Types.AddType(&SVG{}, SVGProps)

// AddNewSVG adds a new svg viewport to given parent node, with given name.
func AddNewSVG(parent ki.Ki, name string) *SVG {
	return parent.AddNewChild(KiT_SVG, name).(*SVG)
}

func (sv *SVG) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*SVG)
	sv.Viewport2D.CopyFieldsFrom(&fr.Viewport2D)
	sv.ViewBox = fr.ViewBox
	sv.Norm = fr.Norm
	sv.InvertY = fr.InvertY
	sv.Pnt = fr.Pnt
	sv.Defs.CopyFrom(&fr.Defs)
	sv.Title = fr.Title
	sv.Desc = fr.Desc
	sv.UniqueIds = nil
}

// Paint satisfies the painter interface
func (sv *SVG) Paint() *gist.Paint {
	return &sv.Pnt.Paint
}

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
	sv.SetFlag(int(gi.VpFlagSVG)) // we are an svg type
	sv.Pnt.Defaults()
	sv.Pnt.FontStyle.BgColor.SetColor(gist.White)
}

func (sv *SVG) Size2D(iter int) {
	sv.InitLayout2D()
	if sv.ViewBox.Size != mat32.Vec2Zero {
		sv.LayState.Alloc.Size = sv.ViewBox.Size
	}
	sv.Size2DAddSpace()
}

// SetUnitContext sets the unit context based on size of viewport and parent
// element (from bbox) and then cache everything out in terms of raw pixel
// dots for rendering -- call at start of render
func SetUnitContext(pc *gist.Paint, vp *gi.Viewport2D, el mat32.Vec2) {
	pc.UnContext.Defaults()
	if vp != nil {
		pc.UnContext.DPI = 96 // paint (SVG) context is always 96 = 1to1
		// if vp.Win != nil {
		// 	pc.UnContext.DPI = vp.Win.LogicalDPI()
		// }
		if vp.Render.Image != nil {
			sz := vp.Render.Image.Bounds().Size()
			pc.UnContext.SetSizes(float32(sz.X), float32(sz.Y), el.X, el.Y)
		} else {
			pc.UnContext.SetSizes(0, 0, el.X, el.Y)
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
	sv.StyleMu.RLock()
	defer sv.StyleMu.RUnlock()

	val := url[4:]
	val = strings.TrimPrefix(strings.TrimSuffix(val, ")"), "#")
	def := sv.FindDefByName(val)
	if def != nil {
		if grad, ok := def.(*gi.Gradient); ok {
			return &grad.Grad
		}
	}
	if sv.CurStyleNode == nil {
		return nil
	}
	ne := sv.CurStyleNode.FindNamedElement(val)
	if grad, ok := ne.(*gi.Gradient); ok {
		return &grad.Grad
	}
	return nil
}

func (sv *SVG) StyleSVG() {
	sv.StyMu.Lock()

	hasTempl, saveTempl := sv.Sty.FromTemplate()
	if !hasTempl || saveTempl {
		sv.Style2DWidget()
	}
	if hasTempl && saveTempl {
		sv.Sty.SaveTemplate()
	}
	sv.Pnt.Defaults()
	sv.StyMu.Unlock()
	StyleSVG(sv.This().(gi.Node2D))
	SetUnitContext(&sv.Pnt.Paint, sv.AsViewport2D(), sv.ViewBox.Size) // context is viewbox
}

func (sv *SVG) Style2D() {
	sv.StyleSVG()
	sv.StyMu.Lock()
	sv.LayState.SetFromStyle(&sv.Sty.Layout) // also does reset
	sv.StyMu.Unlock()
	if nv, err := sv.PropTry("norm"); err == nil {
		sv.Norm, _ = kit.ToBool(nv)
	}
	if iv, err := sv.PropTry("invert-y"); err == nil {
		sv.InvertY, _ = kit.ToBool(iv)
	}
}

func (sv *SVG) Layout2D(parBBox image.Rectangle, iter int) bool {
	sv.Layout2DBase(parBBox, true, iter)
	// do not call layout on children -- they don't do it
	// this is too late to affect anything
	// svg.Pnt.SetUnitContext(svg.AsViewport2D(), svg.ViewBox.Size)
	return false
}

func (sv *SVG) ConnectEvents2D() {
	// nothing here by default, but subtypes can do things here
}

// IsRendering returns true if the SVG is currently rendering
func (sv *SVG) IsRendering() bool {
	return sv.HasFlag(int(Rendering))
}

func (sv *SVG) Render2D() {
	if sv.PushBounds() {
		sv.SetFlag(int(Rendering))
		sv.This().(gi.Node2D).ConnectEvents2D()
		rs := &sv.Render
		if sv.Fill {
			sv.FillViewport()
		}
		if sv.Norm {
			sv.SetNormXForm()
		}
		rs.PushXForm(sv.Pnt.XForm)
		sv.Render2DChildren() // we must do children first, then us!
		sv.PopBounds()
		rs.PopXForm()
		sv.RenderViewport2D() // update our parent image
		sv.ClearFlag(int(Rendering))
	}
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
	sv.FuncDownMeFirst(0, nil, func(k ki.Ki, level int, d interface{}) bool {
		sv.NodeEnsureUniqueId(k)
		return ki.Continue
	})
}

// NodeEnsureUniqueId ensures that the given node has a unique Id
// Call this on any newly-created nodes.
func (sv *SVG) NodeEnsureUniqueId(kn ki.Ki) {
	elnm := ""
	svi, issvi := kn.(NodeSVG)
	if issvi {
		elnm = svi.SVGName()
		// } else if gr, ok := kn.(*gi.Gradient); ok { // don't need gradients to be unique
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

// SVGFlags extend gi.VpFlags to hold SVG node state
type SVGFlags int

//go:generate stringer -type=SVGFlags

var KiT_SVGFlags = kit.Enums.AddEnumExt(gi.KiT_VpFlags, SVGFlagsN, kit.BitFlag, nil)

const (
	// Rendering means that the SVG is currently redrawing
	// Can be useful to check for animations etc to decide whether to
	// drive another update
	Rendering SVGFlags = SVGFlags(gi.VpFlagsN) + iota

	SVGFlagsN
)

var SVGProps = ki.Props{
	"EnumType:Flag": KiT_SVGFlags,
	"ToolBar": ki.PropSlice{
		{"OpenXML", ki.Props{
			"label": "Open...",
			"desc":  "Open SVG XML-formatted file",
			"icon":  "file-open",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"ext": ".svg",
				}},
			},
		}},
		{"SaveXML", ki.Props{
			"label": "SaveAs...",
			"desc":  "Save SVG content to an XML-formatted file.",
			"icon":  "file-save",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"ext": ".svg",
				}},
			},
		}},
	},
}
