// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"strconv"
	"strings"

	"goki.dev/colors"
	"goki.dev/girl/abilities"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi"
	"goki.dev/goosi/events"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

// Config notes: only needs config when number of kids changes
// otherwise just needs new layout

// Splits allocates a fixed proportion of space to each child, along given
// dimension, always using only the available space given to it by its parent
// (i.e., it will force its children, which should be layouts (typically
// Frame's), to have their own scroll bars as necessary).  It should
// generally be used as a main outer-level structure within a window,
// providing a framework for inner elements -- it allows individual child
// elements to update independently and thus is important for speeding update
// performance.  It uses the Widget Parts to hold the splitter widgets
// separately from the children that contain the rest of the scenegraph to be
// displayed within each region.
//
//goki:embedder
type Splits struct {
	WidgetBase

	// size of the handle region in the middle of each split region, where the splitter can be dragged -- other-dimension size is 2x of this
	HandleSize units.Value `xml:"handle-size" desc:"size of the handle region in the middle of each split region, where the splitter can be dragged -- other-dimension size is 2x of this"`

	// proportion (0-1 normalized, enforced) of space allocated to each element -- can enter 0 to collapse a given element
	Splits []float32 `desc:"proportion (0-1 normalized, enforced) of space allocated to each element -- can enter 0 to collapse a given element"`

	// A saved version of the splits which can be restored -- for dynamic collapse / expand operations
	SavedSplits []float32 `desc:"A saved version of the splits which can be restored -- for dynamic collapse / expand operations"`

	// dimension along which to split the space
	Dim mat32.Dims `desc:"dimension along which to split the space"`
}

func (sl *Splits) CopyFieldsFrom(frm any) {
	fr := frm.(*Splits)
	sl.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	sl.HandleSize = fr.HandleSize
	mat32.CopyFloat32s(&sl.Splits, fr.Splits)
	mat32.CopyFloat32s(&sl.SavedSplits, fr.SavedSplits)
	sl.Dim = fr.Dim
}

func (sl *Splits) OnInit() {
	sl.SplitsHandlers()
	sl.SplitsStyles()
}

func (sl *Splits) SplitsStyles() {
	sl.AddStyles(func(s *styles.Style) {
		sl.HandleSize.SetDp(10)

		s.MaxWidth.SetDp(-1)
		s.MaxHeight.SetDp(-1)
		s.Margin.Set()
		s.Padding.Set()
	})
}

func (sl *Splits) OnChildAdded(child ki.Ki) {
	if sp, ok := child.(*Splitter); ok {
		sp.ThumbSize = sl.HandleSize
		sp.On(events.SlideStop, func(e events.Event) {
			sl.SetSplitAction(sp.SplitterNo, sp.Value)
		})
	}
}

// UpdateSplits updates the splits to be same length as number of children,
// and normalized
func (sl *Splits) UpdateSplits() {
	sz := len(sl.Kids)
	if sz == 0 {
		return
	}
	if sl.Splits == nil || len(sl.Splits) != sz {
		sl.Splits = make([]float32, sz)
	}
	sum := float32(0.0)
	for _, sp := range sl.Splits {
		sum += sp
	}
	if sum == 0 { // set default even splits
		sl.EvenSplits()
		sum = 1.0
	} else {
		norm := 1.0 / sum
		for i := range sl.Splits {
			sl.Splits[i] *= norm
		}
	}
}

// EvenSplits splits space evenly across all panels
func (sl *Splits) EvenSplits() {
	updt := sl.UpdateStart()
	sz := len(sl.Kids)
	if sz == 0 {
		return
	}
	even := 1.0 / float32(sz)
	for i := range sl.Splits {
		sl.Splits[i] = even
	}
	sl.UpdateEndLayout(updt)
}

// SetSplits sets the split proportions -- can use 0 to hide / collapse a
// child entirely.
func (sl *Splits) SetSplits(splits ...float32) {
	sl.UpdateSplits()
	sz := len(sl.Kids)
	mx := min(sz, len(splits))
	for i := 0; i < mx; i++ {
		sl.Splits[i] = splits[i]
	}
	sl.UpdateSplits()
}

// SetSplitsList sets the split proportions using a list (slice) argument,
// instead of variable args -- e.g., for Python or other external users.
// can use 0 to hide / collapse a child entirely -- just does the basic local
// update start / end -- use SetSplitsAction to trigger full rebuild
// which is typically required
func (sl *Splits) SetSplitsList(splits []float32) {
	sl.SetSplits(splits...)
}

// SetSplitsAction sets the split proportions -- can use 0 to hide / collapse a
// child entirely -- does full rebuild at level of scene
func (sl *Splits) SetSplitsAction(splits ...float32) {
	updt := sl.UpdateStart()
	sl.SetSplits(splits...)
	sl.UpdateEndLayout(updt)
}

// SaveSplits saves the current set of splits in SavedSplits, for a later RestoreSplits
func (sl *Splits) SaveSplits() {
	sz := len(sl.Splits)
	if sz == 0 {
		return
	}
	if sl.SavedSplits == nil || len(sl.SavedSplits) != sz {
		sl.SavedSplits = make([]float32, sz)
	}
	copy(sl.SavedSplits, sl.Splits)
}

// RestoreSplits restores a previously-saved set of splits (if it exists), does an update
func (sl *Splits) RestoreSplits() {
	if sl.SavedSplits == nil {
		return
	}
	sl.SetSplitsAction(sl.SavedSplits...)
}

// CollapseChild collapses given child(ren) (sets split proportion to 0),
// optionally saving the prior splits for later Restore function -- does an
// Update -- triggered by double-click of splitter
func (sl *Splits) CollapseChild(save bool, idxs ...int) {
	updt := sl.UpdateStart()
	if save {
		sl.SaveSplits()
	}
	sz := len(sl.Kids)
	for _, idx := range idxs {
		if idx >= 0 && idx < sz {
			sl.Splits[idx] = 0
		}
	}
	sl.UpdateSplits()
	sl.UpdateEndLayout(updt)
}

// RestoreChild restores given child(ren) -- does an Update
func (sl *Splits) RestoreChild(idxs ...int) {
	updt := sl.UpdateStart()
	sz := len(sl.Kids)
	for _, idx := range idxs {
		if idx >= 0 && idx < sz {
			sl.Splits[idx] = 1.0 / float32(sz)
		}
	}
	sl.UpdateSplits()
	sl.UpdateEndLayout(updt)
}

// IsCollapsed returns true if given split number is collapsed
func (sl *Splits) IsCollapsed(idx int) bool {
	sz := len(sl.Kids)
	if idx >= 0 && idx < sz {
		return sl.Splits[idx] < 0.01
	}
	return false
}

// SetSplitAction sets the new splitter value, for given splitter -- new
// value is 0..1 value of position of that splitter -- it is a sum of all the
// positions up to that point.  Splitters are updated to ensure that selected
// position is achieved, while dividing remainder appropriately.
func (sl *Splits) SetSplitAction(idx int, nwval float32) {
	updt := sl.UpdateStart()
	sz := len(sl.Splits)
	oldsum := float32(0)
	for i := 0; i <= idx; i++ {
		oldsum += sl.Splits[i]
	}
	delta := nwval - oldsum
	oldval := sl.Splits[idx]
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
					sl.Splits[i] = dper
				}
			}
		} else {
			for i := idx + 1; i < sz; i++ {
				curval := sl.Splits[i]
				sl.Splits[i] = rmdr * (curval / oldrmdr) // proportional
			}
		}
	}
	sl.Splits[idx] = uval
	// fmt.Printf("splits: %v value: %v  splts: %v\n", idx, nwval, sl.Splits)
	sl.UpdateSplits()
	// fmt.Printf("splits: %v\n", sl.Splits)
	sl.UpdateEndRender(updt)
}

func (sl *Splits) ConfigWidget(sc *Scene) {
	sl.NewParts(LayoutNil)
	sl.UpdateSplits()
	sl.ConfigSplitters(sc)
}

func (sl *Splits) ConfigSplitters(sc *Scene) {
	sz := len(sl.Kids)
	mods, updt := sl.Parts.SetNChildren(sz-1, SplitterType, "Splitter")
	odim := mat32.OtherDim(sl.Dim)
	spc := sl.BoxSpace()
	size := sl.LayState.Alloc.Size.Dim(sl.Dim) - spc.Size().Dim(sl.Dim)
	handsz := sl.HandleSize.Dots
	mid := 0.5 * (sl.LayState.Alloc.Size.Dim(odim) - spc.Size().Dim(odim))
	spicon := icons.DragHandle
	if sl.Dim == mat32.X {
		spicon = icons.DragIndicator
	}
	for i, spk := range *sl.Parts.Children() {
		sp := spk.(*Splitter)
		sp.SplitterNo = i
		sp.Icon = spicon
		sp.Dim = sl.Dim
		sp.LayState.Alloc.Size.SetDim(sl.Dim, size)
		sp.LayState.Alloc.Size.SetDim(odim, handsz*2)
		sp.LayState.Alloc.SizeOrig = sp.LayState.Alloc.Size
		sp.LayState.Alloc.PosRel.SetDim(sl.Dim, 0)
		sp.LayState.Alloc.PosRel.SetDim(odim, mid-handsz+float32(i)*handsz*4)
		sp.LayState.Alloc.PosOrig = sp.LayState.Alloc.PosRel
		sp.Min = 0.0
		sp.Max = 1.0
		sp.Snap = false
		sp.ThumbSize = sl.HandleSize
	}
	if mods {
		sl.Parts.UpdateEnd(updt)
	}
}

func (sl *Splits) SplitsKeys() {
	sl.OnKeyChord(func(e events.Event) {
		kc := string(e.KeyChord())
		mod := "Control+"
		if goosi.TheApp.Platform() == goosi.MacOS {
			mod = "Meta+"
		}
		if !strings.HasPrefix(kc, mod) {
			return
		}
		kns := kc[len(mod):]

		knc, err := strconv.Atoi(kns)
		if err != nil {
			return
		}
		kn := int(knc)
		// fmt.Printf("kc: %v kns: %v kn: %v\n", kc, kns, kn)
		if kn == 0 {
			e.SetHandled()
			sl.EvenSplits()
		} else if kn <= len(sl.Kids) {
			e.SetHandled()
			if sl.Splits[kn-1] <= 0.01 {
				sl.RestoreChild(kn - 1)
			} else {
				sl.CollapseChild(true, kn-1)
			}
		}
	})
}

func (sl *Splits) SplitsHandlers() {
	sl.SplitsKeys()
}

func (sl *Splits) StyleSplits(sc *Scene) {
	sl.ApplyStyleWidget(sc)
	// todo: props?
	// sl.HandleSize.SetFmInheritProp("handle-size", sl.This(), ki.NoInherit, ki.TypeProps)
	// sl.HandleSize.ToDots(&sl.Style.UnContext)
}

func (sl *Splits) ApplyStyle(sc *Scene) {
	sl.StyMu.Lock()

	sl.StyleSplits(sc)
	sl.UpdateSplits()
	sl.StyMu.Unlock()

	sl.ConfigSplitters(sc)
}

func (sl *Splits) DoLayout(sc *Scene, parBBox image.Rectangle, iter int) bool {
	sl.DoLayoutBase(sc, parBBox, iter)
	sl.DoLayoutParts(sc, parBBox, iter)
	sl.UpdateSplits()

	handsz := sl.HandleSize.Dots
	// fmt.Printf("handsz: %v\n", handsz)
	sz := len(sl.Kids)
	odim := mat32.OtherDim(sl.Dim)
	spc := sl.BoxSpace()
	size := sl.LayState.Alloc.Size.Dim(sl.Dim) - spc.Size().Dim(sl.Dim)
	avail := size - handsz*float32(sz-1)
	// fmt.Printf("avail: %v\n", avail)
	osz := sl.LayState.Alloc.Size.Dim(odim) - spc.Size().Dim(odim)
	pos := float32(0.0)

	spsum := float32(0)
	for i, sp := range sl.Splits {
		_, wb := AsWidget(sl.Kids[i])
		if wb == nil {
			continue
		}
		isz := sp * avail
		wb.LayState.Alloc.Size.SetDim(sl.Dim, isz)
		wb.LayState.Alloc.Size.SetDim(odim, osz)
		wb.LayState.Alloc.SizeOrig = wb.LayState.Alloc.Size
		wb.LayState.Alloc.PosRel.SetDim(sl.Dim, pos)
		wb.LayState.Alloc.PosRel.SetDim(odim, spc.Pos().Dim(odim))
		// fmt.Printf("spl: %v sp: %v size: %v alloc: %v  pos: %v\n", i, sp, isz, wb.LayState.Alloc.SizeOrig, wb.LayState.Alloc.PosRel)

		pos += isz + handsz

		spsum += sp
		if i < sz-1 {
			spl := sl.Parts.Child(i).(*Splitter)
			spl.Value = spsum
			spl.UpdatePosFromValue(spl.Value)
		}
	}

	return sl.DoLayoutChildren(sc, iter)
}

func (sl *Splits) Render(sc *Scene) {
	if sl.PushBounds(sc) {
		for i, kid := range sl.Kids {
			wi, wb := AsWidget(kid)
			if wb == nil {
				continue
			}
			sp := sl.Splits[i]
			if sp <= 0.01 {
				wb.SetFlag(true, Invisible)
			} else {
				wb.SetFlag(false, Invisible)
			}
			wi.Render(sc) // needs to disconnect using invisible
		}
		sl.Parts.Render(sc)
		sl.PopBounds(sc)
	}
}

// func (sl *Splits) StateIs(states.Focused) bool {
// 	return sl.ContainsFocus() // anyone within us gives us focus..
// }

////////////////////////////////////////////////////////////////////////////////////////
//    Splitter

// TODO(kai): decide on splitter structure

// Splitter provides the splitter handle and line separating two elements in a
// Splits, with draggable resizing of the splitter -- parent is Parts
// layout of the Splits -- based on Slider
type Splitter struct {
	Slider

	// splitter number this one is
	SplitterNo int `desc:"splitter number this one is"`

	// copy of the win bbox, used for translating mouse events when the bbox is restricted to the slider itself
	OrigWinBBox image.Rectangle `copy:"-" json:"-" xml:"-" desc:"copy of the win bbox, used for translating mouse events when the bbox is restricted to the slider itself"`
}

func (sr *Splitter) OnInit() {
	sr.SplitterHandlers()
	sr.SplitterStyles()
}

func (sr *Splitter) SplitterStyles() {
	// STYTODO: fix splitter styles
	sr.ValThumb = false
	sr.ThumbSize = units.Dp(10) // will be replaced by parent HandleSize
	sr.Step = 0.01
	sr.PageStep = 0.1
	sr.Max = 1.0
	sr.Snap = false
	sr.Prec = 4
	sr.SetFlag(true, InstaDrag)

	sr.AddStyles(func(s *styles.Style) {
		s.Margin.Set()
		s.Padding.Set(units.Dp(6))
		s.BackgroundColor.SetSolid(colors.Scheme.Select.Container)
		s.Color = colors.Scheme.OnBackground
		if sr.Dim == mat32.X {
			s.MinWidth.SetDp(2)
			s.MinHeight.SetDp(100)
			s.Height.SetDp(100)
			s.MaxHeight.SetDp(100)
		} else {
			s.MinHeight.SetDp(2)
			s.MinWidth.SetDp(100)
		}
	})

}

func (sr *Splitter) OnChildAdded(child ki.Ki) {
	w, _ := AsWidget(child)
	switch w.Name() {
	case "icon":
		// w.AddStyles(func(s *styles.Style) {
		// 	s.MaxWidth.SetEm(1)
		// 	s.MaxHeight.SetEm(5)
		// 	s.MinWidth.SetEm(1)
		// 	s.MinHeight.SetEm(5)
		// 	s.Margin.Set()
		// 	s.Padding.Set()
		// 	s.AlignV = styles.AlignMiddle
		// })
	}
}

func (sr *Splitter) ConfigWidget(sc *Scene) {
	sr.ConfigSlider(sc)
	sr.ConfigParts(sc)
}

func (sr *Splitter) ApplyStyle(sc *Scene) {
	sr.SetState(false, abilities.Focusable)
	sr.StyleSlider(sc)
}

func (sr *Splitter) GetSize(sc *Scene, iter int) {
	sr.InitLayout(sc)
}

func (sr *Splitter) DoLayout(sc *Scene, parBBox image.Rectangle, iter int) bool {
	sr.DoLayoutBase(sc, parBBox, iter)
	sr.DoLayoutParts(sc, parBBox, iter)
	// sr.SizeFromAlloc()
	sr.Size = sr.LayState.Alloc.Size.Dim(sr.Dim)
	sr.UpdatePosFromValue(sr.Value)
	sr.SlideStartPos = sr.Pos
	sr.BBoxMu.RLock()
	sr.OrigWinBBox = sr.ScBBox
	sr.BBoxMu.RUnlock()
	return sr.DoLayoutChildren(sc, iter)
}

func (sr *Splitter) PointToRelPos(pt image.Point) image.Point {
	// this updates the SliderPositioner interface to use OrigWinBBox
	return pt.Sub(sr.OrigWinBBox.Min)
}

func (sr *Splitter) UpdateSplitterPos() {
	/*
		spc := sr.BoxSpace()
		handsz := sr.ThumbSize.Dots
		off := 0
		if sr.Dim == mat32.X {
			off = sr.OrigWinBBox.Min.X
		} else {
			off = sr.OrigWinBBox.Min.Y
		}
		sz := handsz
		if !sr.HasFlag(NodeDragging) {
			sz += spc.Size().Dim(sr.Dim)
		}
		pos := off + int(sr.Pos-0.5*sz)
		mxpos := off + int(sr.Pos+0.5*sz)

		// SidesTODO: this is all sketchy

		if sr.HasFlag(NodeDragging) {
			win := sr.ParentRenderWin()
			spnm := "gi.Splitter:" + sr.Name()
			spr, ok := win.SpriteByName(spnm)
			if ok {
				spr.Geom.Pos = image.Point{pos, sr.ObjBBox.Min.Y + int(spc.Top)}
			}
		} else {
			sr.BBoxMu.Lock()

			if sr.Dim == mat32.X {
				sr.ScBBox = image.Rect(pos, sr.ObjBBox.Min.Y+int(spc.Top), mxpos, sr.ObjBBox.Max.Y+int(spc.Bottom))
				sr.WinBBox = image.Rect(pos, sr.ObjBBox.Min.Y+int(spc.Top), mxpos, sr.ObjBBox.Max.Y+int(spc.Bottom))
			} else {
				sr.ScBBox = image.Rect(sr.ObjBBox.Min.X+int(spc.Left), pos, sr.ObjBBox.Max.X+int(spc.Right), mxpos)
				sr.WinBBox = image.Rect(sr.ObjBBox.Min.X+int(spc.Left), pos, sr.ObjBBox.Max.X+int(spc.Right), mxpos)
			}
			sr.BBoxMu.Unlock()
		}
	*/
}

// Splits returns our parent splits
func (sr *Splitter) Splits() *Splits {
	if sr.Par == nil || sr.Par.Parent() == nil {
		return nil
	}
	sli := AsSplits(sr.Par.Parent())
	if sli == nil {
		return nil
	}
	return sli
}

func (sr *Splitter) SplitterMouse() {
	sr.On(events.MouseDown, func(e events.Event) {
		// if srr.IsDisabled() {
		// 	me.SetHandled()
		// 	srr.SetSelected(!srr.StateIs(states.Selected))
		// 	srr.EmitSelectedSignal()
		// 	srr.UpdateSig()
		// } else {
		if e.MouseButton() != events.Left {
			return
		}
		e.SetHandled()
		ed := sr.This().(SliderPositioner).PointToRelPos(e.Pos())
		st := &sr.Style
		// SidesTODO: unsure about dim
		spc := st.TotalMargin().Pos().Dim(sr.Dim) + 0.5*sr.ThSize
		if sr.Dim == mat32.X {
			sr.SetSliderPos(float32(ed.X) - spc)
		} else {
			sr.SetSliderPos(float32(ed.Y) - spc)
		}
	})
	sr.On(events.DoubleClick, func(e events.Event) {
		e.SetHandled()
		sl := sr.Splits()
		if sl != nil {
			if sl.IsCollapsed(sr.SplitterNo) {
				sl.RestoreSplits()
			} else {
				sl.CollapseChild(true, sr.SplitterNo)
			}
		}
	})
	// todo: just disabling at this point to prevent bad side-effects
	// sr.On(events.Scroll, func(e events.Event) {
	// 	if srr.IsInactive() {
	// 		return
	// 	}
	// 	e := d.(*events.Scroll)
	// 	e.SetHandled()
	// 	cur := float32(srr.Pos)
	// 	if srr.Dim == mat32.X {
	// 		srr.SliderMove(cur, cur+float32(e.NonZeroDelta(true))) // preferX
	// 	} else {
	// 		srr.SliderMove(cur, cur-float32(e.NonZeroDelta(false))) // preferY
	// 	}
	// })
}

func (sr *Splitter) SplitterHandlers() {
	sr.SliderMouse()
	sr.SliderKeys()
	sr.SplitterMouse()
}

func (sr *Splitter) Render(sc *Scene) {
	/*
		win := sr.ParentRenderWin()
		spnm := "gi.Splitter:" + sr.Name()
		if sr.HasFlag(NodeDragging) {
			ick := sr.Parts.ChildByType(IconType, ki.Embeds, 0)
			if ick == nil {
				return
			}
			ic := ick.(*Icon)
			spr, ok := win.SpriteByName(spnm)
			if !ok {
				// spr = NewSprite(spnm, image.Point{}, sr.ScBBox.Min)
				// spr.GrabRenderFrom(ic)
				// win.AddSprite(spr)
				// win.ActivateSprite(spnm)
			}
			sr.UpdateSplitterPos()
			win.UpdateSig()
		} else {
			if win.DeleteSprite(spnm) {
				win.UpdateSig()
			}
			sr.UpdateSplitterPos()
			if sr.PushBounds(sc) {
				sr.RenderSplitter(sc)
				sr.RenderChildren(sc)
				sr.PopBounds(sc)
			}
		}
	*/
}

// RenderSplitter does the default splitter rendering
func (sr *Splitter) RenderSplitter(sc *Scene) {
	sr.UpdateSplitterPos()

	if sr.Icon.IsValid() && sr.Parts.HasChildren() {
		sr.Parts.Render(sc)
	}
	// else {
	rs, pc, st := sr.RenderLock(sc)

	pc.StrokeStyle.SetColor(nil)
	pc.FillStyle.SetFullColor(&st.BackgroundColor)

	// pos := mat32.NewVec2FmPoint(sr.ScBBox.Min)
	// pos.SetSubDim(mat32.OtherDim(sr.Dim), 10.0)
	// sz := mat32.NewVec2FmPoint(sr.ScBBox.Size())
	// sr.RenderBoxImpl(pos, sz, st.Border)

	sr.RenderStdBox(sc, st)

	sr.RenderUnlock(rs)
	// }
}
