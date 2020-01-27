// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"strconv"
	"strings"

	"github.com/goki/gi/mat32"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/oswin/mouse"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
//    SplitView

// SplitView allocates a fixed proportion of space to each child, along given
// dimension, always using only the available space given to it by its parent
// (i.e., it will force its children, which should be layouts (typically
// Frame's), to have their own scroll bars as necessary).  It should
// generally be used as a main outer-level structure within a window,
// providing a framework for inner elements -- it allows individual child
// elements to update independently and thus is important for speeding update
// performance.  It uses the Widget Parts to hold the splitter widgets
// separately from the children that contain the rest of the scenegraph to be
// displayed within each region.
type SplitView struct {
	PartsWidgetBase
	HandleSize  units.Value `xml:"handle-size" desc:"size of the handle region in the middle of each split region, where the splitter can be dragged -- other-dimension size is 2x of this"`
	Splits      []float32   `desc:"proportion (0-1 normalized, enforced) of space allocated to each element -- can enter 0 to collapse a given element"`
	SavedSplits []float32   `desc:"A saved version of the splits which can be restored -- for dynamic collapse / expand operations"`
	Dim         mat32.Dims  `desc:"dimension along which to split the space"`
}

var KiT_SplitView = kit.Types.AddType(&SplitView{}, SplitViewProps)

// AddNewSplitView adds a new splitview to given parent node, with given name.
func AddNewSplitView(parent ki.Ki, name string) *SplitView {
	return parent.AddNewChild(KiT_SplitView, name).(*SplitView)
}

func (nb *SplitView) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*SplitView)
	nb.PartsWidgetBase.CopyFieldsFrom(&fr.PartsWidgetBase)
	nb.HandleSize = fr.HandleSize
	mat32.CopyFloat32s(&nb.Splits, fr.Splits)
	mat32.CopyFloat32s(&nb.SavedSplits, fr.SavedSplits)
	nb.Dim = fr.Dim
}

var SplitViewProps = ki.Props{
	"EnumType:Flag": KiT_NodeFlags,
	"handle-size":   units.NewPx(10),
	"max-width":     -1.0,
	"max-height":    -1.0,
	"margin":        0,
	"padding":       0,
}

// UpdateSplits updates the splits to be same length as number of children,
// and normalized
func (sv *SplitView) UpdateSplits() {
	sz := len(sv.Kids)
	if sz == 0 {
		return
	}
	if sv.Splits == nil || len(sv.Splits) != sz {
		sv.Splits = make([]float32, sz)
	}
	sum := float32(0.0)
	for _, sp := range sv.Splits {
		sum += sp
	}
	if sum == 0 { // set default even splits
		sv.EvenSplits()
		sum = 1.0
	} else {
		norm := 1.0 / sum
		for i := range sv.Splits {
			sv.Splits[i] *= norm
		}
	}
}

// EvenSplits splits space evenly across all panels
func (sv *SplitView) EvenSplits() {
	sz := len(sv.Kids)
	if sz == 0 {
		return
	}
	even := 1.0 / float32(sz)
	for i := range sv.Splits {
		sv.Splits[i] = even
	}
}

// SetSplits sets the split proportions -- can use 0 to hide / collapse a
// child entirely -- just does the basic local update start / end
// use SetSplitsAction to trigger full rebuild which is typically required
func (sv *SplitView) SetSplits(splits ...float32) {
	updt := sv.UpdateStart()
	sv.UpdateSplits()
	sz := len(sv.Kids)
	mx := ints.MinInt(sz, len(splits))
	for i := 0; i < mx; i++ {
		sv.Splits[i] = splits[i]
	}
	sv.UpdateSplits()
	sv.UpdateEnd(updt)
}

// SetSplitsList sets the split proportions using a list (slice) argument,
// instead of variable args -- e.g., for Python or other external users.
// can use 0 to hide / collapse a child entirely -- just does the basic local
// update start / end -- use SetSplitsAction to trigger full rebuild
// which is typically required
func (sv *SplitView) SetSplitsList(splits []float32) {
	sv.SetSplits(splits...)
}

// SetSplitsAction sets the split proportions -- can use 0 to hide / collapse a
// child entirely -- does full rebuild at level of viewport
func (sv *SplitView) SetSplitsAction(splits ...float32) {
	sv.SetSplits(splits...)
	// sv.WinFullReRender() // tell window to do a full redraw
	sv.Viewport.SetNeedsFullRender()
}

// SaveSplits saves the current set of splits in SavedSplits, for a later RestoreSplits
func (sv *SplitView) SaveSplits() {
	sz := len(sv.Splits)
	if sz == 0 {
		return
	}
	if sv.SavedSplits == nil || len(sv.SavedSplits) != sz {
		sv.SavedSplits = make([]float32, sz)
	}
	copy(sv.SavedSplits, sv.Splits)
}

// RestoreSplits restores a previously-saved set of splits (if it exists), does an update
func (sv *SplitView) RestoreSplits() {
	if sv.SavedSplits == nil {
		return
	}
	sv.SetSplitsAction(sv.SavedSplits...)
}

// CollapseChild collapses given child(ren) (sets split proportion to 0),
// optionally saving the prior splits for later Restore function -- does an
// Update -- triggered by double-click of splitter
func (sv *SplitView) CollapseChild(save bool, idxs ...int) {
	updt := sv.UpdateStart()
	if save {
		sv.SaveSplits()
	}
	sz := len(sv.Kids)
	for _, idx := range idxs {
		if idx >= 0 && idx < sz {
			sv.Splits[idx] = 0
		}
	}
	sv.UpdateSplits()
	sv.Viewport.SetNeedsFullRender() // splits typically require full rebuild
	sv.UpdateEnd(updt)
}

// RestoreChild restores given child(ren) -- does an Update
func (sv *SplitView) RestoreChild(idxs ...int) {
	updt := sv.UpdateStart()
	sz := len(sv.Kids)
	for _, idx := range idxs {
		if idx >= 0 && idx < sz {
			sv.Splits[idx] = 1.0 / float32(sz)
		}
	}
	sv.UpdateSplits()
	sv.Viewport.SetNeedsFullRender() // splits typically require full rebuild
	sv.UpdateEnd(updt)
}

// IsCollapsed returns true if given split number is collapsed
func (sv *SplitView) IsCollapsed(idx int) bool {
	sz := len(sv.Kids)
	if idx >= 0 && idx < sz {
		return sv.Splits[idx] < 0.01
	}
	return false
}

// SetSplitAction sets the new splitter value, for given splitter -- new
// value is 0..1 value of position of that splitter -- it is a sum of all the
// positions up to that point.  Splitters are updated to ensure that selected
// position is achieved, while dividing remainder appropriately.
func (sv *SplitView) SetSplitAction(idx int, nwval float32) {
	sz := len(sv.Splits)
	oldsum := float32(0)
	for i := 0; i <= idx; i++ {
		oldsum += sv.Splits[i]
	}
	delta := nwval - oldsum
	oldval := sv.Splits[idx]
	uval := oldval + delta
	if uval < 0 {
		uval = 0
		delta = -oldval
		nwval = oldsum + delta
	}
	rmdr := 1 - nwval
	if idx < sz-1 {
		oldrmdr := 1 - oldsum
		if oldrmdr <= 0 {
			if rmdr > 0 {
				dper := rmdr / float32((sz-1)-idx)
				for i := idx + 1; i < sz; i++ {
					sv.Splits[i] = dper
				}
			}
		} else {
			for i := idx + 1; i < sz; i++ {
				curval := sv.Splits[i]
				sv.Splits[i] = rmdr * (curval / oldrmdr) // proportional
			}
		}
	}
	sv.Splits[idx] = uval
	// fmt.Printf("splits: %v value: %v  splts: %v\n", idx, nwval, sv.Splits)
	sv.UpdateSplits()
	// fmt.Printf("splits: %v\n", sv.Splits)
	sv.Viewport.SetNeedsFullRender() // splits typically require full rebuild
}

func (sv *SplitView) Init2D() {
	sv.Parts.Lay = LayoutNil
	sv.Init2DWidget()
	sv.UpdateSplits()
	sv.ConfigSplitters()
}

func (sv *SplitView) ConfigSplitters() {
	sz := len(sv.Kids)
	mods, updt := sv.Parts.SetNChildren(sz-1, KiT_Splitter, "Splitter")
	odim := mat32.OtherDim(sv.Dim)
	spc := sv.Sty.BoxSpace()
	size := sv.LayData.AllocSize.Dim(sv.Dim) - 2*spc
	handsz := sv.HandleSize.Dots
	mid := 0.5 * (sv.LayData.AllocSize.Dim(odim) - 2*spc)
	spicon := IconName("")
	if sv.Dim == mat32.X {
		spicon = IconName("handle-circles-vert")
	} else {
		spicon = IconName("handle-circles-horiz")
	}
	for i, spk := range *sv.Parts.Children() {
		sp := spk.(*Splitter)
		sp.Defaults()
		sp.SplitterNo = i
		sp.Icon = spicon
		sp.Dim = sv.Dim
		sp.LayData.AllocSize.SetDim(sv.Dim, size)
		sp.LayData.AllocSize.SetDim(odim, handsz*2)
		sp.LayData.AllocSizeOrig = sp.LayData.AllocSize
		sp.LayData.AllocPosRel.SetDim(sv.Dim, 0)
		sp.LayData.AllocPosRel.SetDim(odim, mid-handsz+float32(i)*handsz*4)
		sp.LayData.AllocPosOrig = sp.LayData.AllocPosRel
		sp.Min = 0.0
		sp.Max = 1.0
		sp.Snap = false
		sp.SetProp("thumb-size", sv.HandleSize)
		sp.ThumbSize = sv.HandleSize
		if mods {
			sp.SliderSig.ConnectOnly(sv.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(SliderReleased) {
					spr, _ := recv.Embed(KiT_SplitView).(*SplitView)
					spl := send.(*Splitter)
					spr.SetSplitAction(spl.SplitterNo, spl.Value)
				}
			})
		}
	}
	if mods {
		sv.Parts.UpdateEnd(updt)
	}
}

func (sv *SplitView) KeyInput(kt *key.ChordEvent) {
	kc := string(kt.Chord())
	mod := "Control+"
	if oswin.TheApp.Platform() == oswin.MacOS {
		mod = "Meta+"
	}
	if !strings.HasPrefix(kc, mod) {
		return
	}
	kns := kc[len(mod):]

	knc, err := strconv.ParseInt(kns, 10, 64)
	if err != nil {
		return
	}
	kn := int(knc)
	// fmt.Printf("kc: %v kns: %v kn: %v\n", kc, kns, kn)
	if kn == 0 {
		sv.EvenSplits()
		sv.SetFullReRender()
		sv.UpdateSig()
		kt.SetProcessed()
	} else if kn <= len(sv.Kids) {
		sv.SetFullReRender()
		if sv.Splits[kn-1] <= 0.01 {
			sv.RestoreChild(kn - 1)
		} else {
			sv.CollapseChild(true, kn-1)
		}
		kt.SetProcessed()
	}
}

func (sv *SplitView) KeyChordEvent() {
	sv.ConnectEvent(oswin.KeyChordEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		svv := recv.Embed(KiT_SplitView).(*SplitView)
		svv.KeyInput(d.(*key.ChordEvent))
	})
}

func (sv *SplitView) SplitViewEvents() {
	sv.KeyChordEvent()
}

func (sv *SplitView) StyleSplitView() {
	sv.Style2DWidget()
	sv.LayData.SetFromStyle(&sv.Sty.Layout) // also does reset
	sv.HandleSize.SetFmInheritProp("handle-size", sv.This(), ki.NoInherit, ki.TypeProps)
	sv.HandleSize.ToDots(&sv.Sty.UnContext)
}

func (sv *SplitView) Style2D() {
	sv.StyleSplitView()
	sv.LayData.SetFromStyle(&sv.Sty.Layout) // also does reset
	sv.UpdateSplits()
	sv.ConfigSplitters()
}

func (sv *SplitView) Layout2D(parBBox image.Rectangle, iter int) bool {
	sv.ConfigSplitters()
	sv.Layout2DBase(parBBox, true, iter) // init style
	sv.Layout2DParts(parBBox, iter)
	sv.UpdateSplits()

	handsz := sv.HandleSize.Dots
	// fmt.Printf("handsz: %v\n", handsz)
	sz := len(sv.Kids)
	odim := mat32.OtherDim(sv.Dim)
	spc := sv.Sty.BoxSpace()
	size := sv.LayData.AllocSize.Dim(sv.Dim) - 2*spc
	avail := size - handsz*float32(sz-1)
	// fmt.Printf("avail: %v\n", avail)
	osz := sv.LayData.AllocSize.Dim(odim) - 2*spc
	pos := float32(0.0)

	spsum := float32(0)
	for i, sp := range sv.Splits {
		gis := sv.Kids[i].(Node2D).AsWidget()
		if gis == nil {
			continue
		}
		if gis.TypeEmbeds(KiT_Frame) {
			gis.SetReRenderAnchor()
		}
		isz := sp * avail
		gis.LayData.AllocSize.SetDim(sv.Dim, isz)
		gis.LayData.AllocSize.SetDim(odim, osz)
		gis.LayData.AllocSizeOrig = gis.LayData.AllocSize
		gis.LayData.AllocPosRel.SetDim(sv.Dim, pos)
		gis.LayData.AllocPosRel.SetDim(odim, spc)
		// fmt.Printf("spl: %v sp: %v size: %v alloc: %v  pos: %v\n", i, sp, isz, gis.LayData.AllocSizeOrig, gis.LayData.AllocPosRel)

		pos += isz + handsz

		spsum += sp
		if i < sz-1 {
			spl := sv.Parts.Child(i).(*Splitter)
			spl.Value = spsum
			spl.UpdatePosFromValue()
		}
	}

	return sv.Layout2DChildren(iter)
}

func (sv *SplitView) Render2D() {
	if sv.FullReRenderIfNeeded() {
		return
	}
	if sv.PushBounds() {
		sv.This().(Node2D).ConnectEvents2D()
		for i, kid := range sv.Kids {
			nii, ni := KiToNode2D(kid)
			if nii != nil {
				sp := sv.Splits[i]
				if sp <= 0.01 {
					ni.SetInvisible()
				} else {
					ni.ClearInvisible()
				}
				nii.Render2D() // needs to disconnect using invisible
			}
		}
		sv.Parts.Render2DTree()
		sv.PopBounds()
	}
}

func (sv *SplitView) ConnectEvents2D() {
	sv.SplitViewEvents()
}

func (sv *SplitView) HasFocus2D() bool {
	return sv.ContainsFocus() // anyone within us gives us focus..
}

////////////////////////////////////////////////////////////////////////////////////////
//    Splitter

// Splitter provides the splitter handle and line separating two elements in a
// SplitView, with draggable resizing of the splitter -- parent is Parts
// layout of the SplitView -- based on SliderBase
type Splitter struct {
	SliderBase
	SplitterNo  int             `desc:"splitter number this one is"`
	OrigWinBBox image.Rectangle `copy:"-" json:"-" xml:"-" desc:"copy of the win bbox, used for translating mouse events when the bbox is restricted to the slider itself"`
}

var KiT_Splitter = kit.Types.AddType(&Splitter{}, SplitterProps)

var SplitterProps = ki.Props{
	"EnumType:Flag":    KiT_NodeFlags,
	"padding":          units.NewPx(6),
	"margin":           units.NewPx(0),
	"background-color": &Prefs.Colors.Background,
	"color":            &Prefs.Colors.Font,
	"#icon": ki.Props{
		"max-width":      units.NewEm(1),
		"max-height":     units.NewEm(5),
		"min-width":      units.NewEm(1),
		"min-height":     units.NewEm(5),
		"margin":         units.NewPx(0),
		"padding":        units.NewPx(0),
		"vertical-align": AlignMiddle,
		"fill":           &Prefs.Colors.Icon,
		"stroke":         &Prefs.Colors.Font,
	},
	SliderSelectors[SliderActive]: ki.Props{},
	SliderSelectors[SliderInactive]: ki.Props{
		"border-color": "highlight-50",
		"color":        "highlight-50",
	},
	SliderSelectors[SliderHover]: ki.Props{
		"background-color": "highlight-10",
	},
	SliderSelectors[SliderFocus]: ki.Props{
		"border-width":     units.NewPx(2),
		"background-color": "samelight-50",
	},
	SliderSelectors[SliderDown]: ki.Props{},
	SliderSelectors[SliderValue]: ki.Props{
		"border-color":     &Prefs.Colors.Icon,
		"background-color": &Prefs.Colors.Icon,
	},
	SliderSelectors[SliderBox]: ki.Props{
		"border-color":     &Prefs.Colors.Background,
		"background-color": &Prefs.Colors.Background,
	},
}

func (sr *Splitter) Defaults() {
	sr.ValThumb = false
	sr.ThumbSize = units.NewPx(10) // will be replaced by parent HandleSize
	sr.Step = 0.01
	sr.PageStep = 0.1
	sr.Max = 1.0
	sr.Snap = false
	sr.Prec = 4
	sr.SetFlag(int(InstaDrag))
}

func (sr *Splitter) Init2D() {
	sr.Init2DSlider()
	sr.Defaults()
	sr.ConfigParts()
}

func (sr *Splitter) ConfigPartsIfNeeded(render bool) {
	if sr.PartsNeedUpdateIconLabel(string(sr.Icon), "") {
		sr.ConfigParts()
	}
	if !sr.Icon.IsValid() || !sr.Parts.HasChildren() {
		return
	}
	ick := sr.Parts.ChildByType(KiT_Icon, ki.Embeds, 0)
	if ick == nil {
		return
	}
	ic := ick.(*Icon)
	handsz := sr.ThumbSize.Dots
	spc := sr.Sty.BoxSpace()
	odim := mat32.OtherDim(sr.Dim)
	sr.LayData.AllocSize.SetDim(odim, 2*(handsz+2*spc))
	sr.LayData.AllocSizeOrig = sr.LayData.AllocSize

	ic.LayData.AllocSize.SetDim(odim, 2*handsz)
	ic.LayData.AllocSize.SetDim(sr.Dim, handsz)
	ic.LayData.AllocPosRel.SetDim(sr.Dim, sr.Pos-(0.5*(handsz+spc)))
	ic.LayData.AllocPosRel.SetDim(odim, 0)
	if render {
		ic.Layout2DTree()
	}
}

func (sr *Splitter) Style2D() {
	sr.ClearFlag(int(CanFocus))
	sr.Style2DWidget()
	sr.LayData.SetFromStyle(&sr.Sty.Layout) // also does reset
	pst := &(sr.Par.(Node2D).AsWidget().Sty)
	for i := 0; i < int(SliderStatesN); i++ {
		sr.StateStyles[i].CopyFrom(&sr.Sty)
		sr.StateStyles[i].SetStyleProps(pst, sr.StyleProps(SliderSelectors[i]), sr.Viewport)
		sr.StateStyles[i].CopyUnitContext(&sr.Sty.UnContext)
	}
	SliderFields.Style(sr, nil, sr.Props, sr.Viewport)
	SliderFields.ToDots(sr, &sr.Sty.UnContext)
	sr.ThSize = sr.ThumbSize.Dots
	sr.ConfigParts()
}

func (sr *Splitter) Size2D(iter int) {
	sr.InitLayout2D()
	if sr.ThSize == 0.0 {
		sr.Defaults()
	}
}

func (sr *Splitter) Layout2D(parBBox image.Rectangle, iter int) bool {
	sr.ConfigPartsIfNeeded(false)
	sr.Layout2DBase(parBBox, true, iter) // init style
	sr.Layout2DParts(parBBox, iter)
	// sr.SizeFromAlloc()
	sr.Size = sr.LayData.AllocSize.Dim(sr.Dim)
	sr.UpdatePosFromValue()
	sr.DragPos = sr.Pos
	sr.OrigWinBBox = sr.WinBBox
	return sr.Layout2DChildren(iter)
}

func (sr *Splitter) PointToRelPos(pt image.Point) image.Point {
	// this updates the SliderPositioner interface to use OrigWinBBox
	return pt.Sub(sr.OrigWinBBox.Min)
}

func (sr *Splitter) UpdateSplitterPos() {
	spc := sr.Sty.BoxSpace()
	ispc := int(spc)
	handsz := sr.ThumbSize.Dots
	off := 0
	if sr.Dim == mat32.X {
		off = sr.OrigWinBBox.Min.X
	} else {
		off = sr.OrigWinBBox.Min.Y
	}
	sz := handsz
	if !sr.IsDragging() {
		sz += 2 * spc
	}
	pos := off + int(sr.Pos-0.5*sz)
	mxpos := off + int(sr.Pos+0.5*sz)

	if sr.IsDragging() {
		win := sr.Viewport.Win
		spnm := "gi.Splitter:" + sr.UniqueName()
		spr, ok := win.SpriteByName(spnm)
		if ok {
			spr.Geom.Pos = image.Point{pos, sr.ObjBBox.Min.Y + ispc}
		}
	} else {
		if sr.Dim == mat32.X {
			sr.VpBBox = image.Rect(pos, sr.ObjBBox.Min.Y+ispc, mxpos, sr.ObjBBox.Max.Y+ispc)
			sr.WinBBox = image.Rect(pos, sr.ObjBBox.Min.Y+ispc, mxpos, sr.ObjBBox.Max.Y+ispc)
		} else {
			sr.VpBBox = image.Rect(sr.ObjBBox.Min.X+ispc, pos, sr.ObjBBox.Max.X+ispc, mxpos)
			sr.WinBBox = image.Rect(sr.ObjBBox.Min.X+ispc, pos, sr.ObjBBox.Max.X+ispc, mxpos)
		}
	}
}

// SplitView returns our parent split view
func (sr *Splitter) SplitView() *SplitView {
	if sr.Par == nil || sr.Par.Parent() == nil {
		return nil
	}
	svi := sr.Par.Parent().Embed(KiT_SplitView)
	if svi == nil {
		return nil
	}
	return svi.(*SplitView)
}

func (sr *Splitter) MouseEvent() {
	sr.ConnectEvent(oswin.MouseEvent, RegPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		me := d.(*mouse.Event)
		srr := recv.Embed(KiT_Splitter).(*Splitter)
		if srr.IsInactive() {
			me.SetProcessed()
			srr.SetSelectedState(!srr.IsSelected())
			srr.EmitSelectedSignal()
			srr.UpdateSig()
		} else {
			if me.Button == mouse.Left {
				me.SetProcessed()
				if me.Action == mouse.Press {
					ed := srr.This().(SliderPositioner).PointToRelPos(me.Where)
					st := &srr.Sty
					spc := st.Layout.Margin.Dots + 0.5*srr.ThSize
					if srr.Dim == mat32.X {
						srr.SliderPress(float32(ed.X) - spc)
					} else {
						srr.SliderPress(float32(ed.Y) - spc)
					}
				} else if me.Action == mouse.DoubleClick {
					sv := srr.SplitView()
					if sv != nil {
						if sv.IsCollapsed(srr.SplitterNo) {
							sv.RestoreSplits()
						} else {
							sv.CollapseChild(true, srr.SplitterNo)
						}
					}
				} else {
					srr.SliderRelease()
				}
			}
		}
	})
}

func (sr *Splitter) SplitterEvents() {
	sr.MouseDragEvent()
	sr.MouseEvent()
	sr.MouseFocusEvent()
	sr.MouseScrollEvent()
	sr.KeyChordEvent()
}

func (sr *Splitter) ConnectEvents2D() {
	sr.SplitterEvents()
}

func (sr *Splitter) Render2D() {
	win := sr.Viewport.Win
	sr.This().(Node2D).ConnectEvents2D()
	spnm := "gi.Splitter:" + sr.UniqueName()
	if sr.IsDragging() {
		ick := sr.Parts.ChildByType(KiT_Icon, ki.Embeds, 0)
		if ick == nil {
			return
		}
		ic := ick.(*Icon)
		icvp := ic.ChildByType(KiT_Viewport2D, ki.Embeds, 0)
		if icvp == nil {
			return
		}
		spr, ok := win.SpriteByName(spnm)
		if !ok {
			spr = win.AddNewSprite(spnm, image.ZP, sr.VpBBox.Min)
			spr.GrabRenderFrom(icvp.(Node2D))
			win.ActivateSprite(spnm)
		}
		sr.UpdateSplitterPos()
		win.RenderOverlays()
	} else {
		if win.DeleteSprite(spnm) {
			win.RenderOverlays()
		}
		sr.UpdateSplitterPos()
		if sr.FullReRenderIfNeeded() {
			return
		}
		if sr.PushBounds() {
			sr.Render2DDefaultStyle()
			sr.Render2DChildren()
			sr.PopBounds()
		}
	}
}

// render using a default style if not otherwise styled
func (sr *Splitter) Render2DDefaultStyle() {
	st := &sr.Sty
	rs := &sr.Viewport.Render
	pc := &rs.Paint

	sr.UpdateSplitterPos()
	sr.ConfigPartsIfNeeded(true)

	if sr.Icon.IsValid() && sr.Parts.HasChildren() {
		sr.Parts.Render2DTree()
	} else {
		pc.StrokeStyle.SetColor(nil)
		pc.FillStyle.SetColorSpec(&st.Font.BgColor)

		pos := mat32.NewVec2FmPoint(sr.VpBBox.Min)
		pos.SetSubDim(mat32.OtherDim(sr.Dim), 10.0)
		sz := mat32.NewVec2FmPoint(sr.VpBBox.Size())
		sr.RenderBoxImpl(pos, sz, 0)
	}
}

func (sr *Splitter) FocusChanged2D(change FocusChanges) {
	switch change {
	case FocusLost:
		sr.SetSliderState(SliderActive) // lose any hover state but whatever..
		sr.UpdateSig()
	case FocusGot:
		sr.SetSliderState(SliderFocus)
		sr.EmitFocusedSignal()
		sr.UpdateSig()
	case FocusInactive: // don't care..
	case FocusActive:
	}
}
