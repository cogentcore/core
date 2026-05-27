package main

import (
	"fmt"
	"image"
	"image/color"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

type MathPad struct {
	core.Frame

	// Type is the styling type of the tabs. If it is changed after
	// the tabs are first configured, Update needs to be called on
	// the tabs.
	Type MathPadTypes

	toolbar  *core.Frame
	sentsfrm *MathPadFrame
}

type MathPadTypes int32 //enums:enum

const (
	// StandardTabs indicates to render the standard type
	// of Material Design style tabs.
	StandardMathPad MathPadTypes = iota

	TopToolbarMathPad
	LeftToolbarMathPad
	BottomToolbarMathPad
	RightToolbarMathPad
)

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/core.MathPad", IDName: "mathpad", Doc: "mathpad divide widgets into logical groups and give users the ability\nto math execute.", Embeds: []types.Field{{Name: "MathPadFrame"}}, Fields: []types.Field{{Name: "Type", Doc: "Type is the styling type of the tabs. If it is changed after\nthe tabs are first configured, Update needs to be called on\nthe tabs."}, {Name: "toolbar"}, {Name: "sentsfrm"}}})

// NewMathPad returns a new [MathPad] with the given optional parent:
// Tabs divide widgets into logical groups and give users the ability
// to freely math execute.
func NewMathPad(parent ...tree.Node) *MathPad { return tree.New[MathPad](parent...) }

// SetType sets the [Tabs.Type]:
// Type is the styling type of the tabs. If it is changed after
// the tabs are first configured, Update needs to be called on
// the tabs.
func (t *MathPad) SetType(v MathPadTypes) *MathPad { t.Type = v; return t }

func (mp *MathPad) Init() {
	mp.Frame.Init()
	mp.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Direction = styles.Column
	})

	mp.Maker(func(p *tree.Plan) {
		tree.AddAt(p, "text", func(w *core.Text) {
			w.SetText("ldjsfldj")
		})
		tree.AddAt(p, "frame", func(w *core.Frame) {
			mp.toolbar = w
			ntb := core.NewButton(w).SetIcon(icons.Add).SetText("MathPadFrame as toolbar toolbar btn")
			ntb.SetTooltip("Add a new tab").SetName("mathpad-button")
			ntb.OnClick(func(e events.Event) {
				fmt.Println("toolbar btn clicked")
				sents := &Sentences{}
				sents.sents = append(sents.sents, &Sentence{children: []*SentenceElement{{text: "asdfsf"}, {text: "83488438"}, {text: "poifjdfjd"}}})
				sents.sents = append(sents.sents, &Sentence{children: []*SentenceElement{{text: "jjbgb"}, {text: "644"}, {text: "iijfgf"}}})
				mp.SetSceneData(sents)
				mp.Update()
			})
			ntb1 := core.NewButton(w).SetIcon(icons.Add).SetText("MathPadFrame as toolbar toolbar btn")
			ntb1.SetTooltip("Add a new tab").SetName("mathpad-button")
			ntb1.OnClick(func(e events.Event) {

			})
		})

		tree.AddAt(p, "mathpadframe", func(w *MathPadFrame) {
			mp.sentsfrm = w
			w.Styler(func(s *styles.Style) {
				s.Overflow.Set(styles.OverflowScroll) // have scrollbar
				s.Grow.Set(1, 1)
				s.Direction = styles.Column
			})
			ntb := core.NewButton(w).SetIcon(icons.Add).SetText("MathPadFrame as toolbar toolbar btn")
			ntb.SetTooltip("Add a new tab").SetName("mathpad-button")
			ntb.OnClick(func(e events.Event) {
				fmt.Println("toolbar btn0 clicked")
			})
			ntb1 := core.NewButton(w).SetIcon(icons.Add).SetText("MathPadFrame as toolbar toolbar btn")
			ntb1.SetTooltip("Add a new tab").SetName("mathpad-button")
			ntb1.OnClick(func(e events.Event) {
				fmt.Println("toolbar btn1 clicked")
			})
		})
	})
}

func (mp *MathPad) SetSceneData(sents *Sentences) {
	mp.sentsfrm.SetSceneData(sents)
	mp.sentsfrm.startCursor()
}

type MathPadFrame struct {
	core.Frame

	painter *paint.Painter

	CursorColor image.Image
	CursorPos   image.Point
}

var _ = types.AddType(&types.Type{Name: "github.com/runsys/wa.MathPadFrame", IDName: "mathpadframe", Doc: "mathpad divide widgets into logical groups and give users the ability\nto math execute.", Embeds: []types.Field{{Name: "MathPadFrame"}}, Fields: []types.Field{{Name: "Type", Doc: "Type is the styling type of the tabs. If it is changed after\nthe tabs are first configured, Update needs to be called on\nthe tabs."}}})

const (
	MathPadFrameSpriteName = " MathPadFrameSpriteName"
)

func (mpfr *MathPadFrame) Init() {
	mpfr.Frame.Init()
	mpfr.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Selectable, abilities.Activatable, abilities.Focusable, abilities.Hoverable, abilities.Draggable, abilities.Clickable, abilities.DoubleClickable, abilities.TripleClickable, abilities.ScrollableUnattended, abilities.Scrollable)
		//s.GrowWrap = false
		s.Grow.Set(1, 1)
		s.Direction = styles.Column
		s.Grow.X = 1
		s.Grow.Y = 1
		s.Border.Radius = styles.BorderRadiusFull
		s.Gap.Zero()
		s.Align.Content = styles.Start
		s.Align.Items = styles.Start
		s.Overflow.Set(styles.OverflowAuto)
		mpfr.CursorColor = colors.Scheme.Primary.Base
	})
	mpfr.OnClick(func(e events.Event) {
		fmt.Println("mathpadframe clicked")
		if !mpfr.IsReadOnly() {
			mpfr.SetFocus()
		}
		switch e.MouseButton() {
		case events.Left:
			mpfr.ForWidgetChildren(func(i int, cw core.Widget, cwb *core.WidgetBase) bool {
				fmt.Println("cwb pos", cwb.Geom.Pos, cwb.Geom.RelPos, e.Pos())
				// if cwb.Geom.Pos.Content.Y-mpfr.Geom.Scroll.Y>=0 && cwb.Geom.Pos.Content.Y-mpfr.Geom.Scroll.Y<mpfr.SceneSize().Y {
				// 	if cwb.Geom.Pos.Content.ToPoint().Y-mpfr.Geom.Scroll.ToPoint().Y>=e.Pos().Y && e.Pos().Y<cwb.Geom.Pos.Content.ToPoint().Y-mpfr.Geom.Scroll.ToPoint().Y+cwb.Geom.Size.Internal.Y {
				// 		cwb.ForWidgetChildren(func(i int, cw1 core.Widget, cwb1 *core.WidgetBase) bool {
				// 			if cwb1.Geom.Pos.Content.X>e.Pos().X && e.Pos().X<cwb1.Geom.Pos.Content.X+cwb1.Geom.Size.Internal.Y {
				// 				mpfr.CursorPos=cwb1.Geom.RelPos.ToPoint().Add(cwb.Geom.RelPos.ToPoint())
				// 				return false
				// 			}
				// 			fmt.Println("cwb pos", cwb.Geom.Pos, cwb.Geom.RelPos, e.Pos())
				// 			return true
				// 		})
				// 		return false
				// 	}
				// }
				// fmt.Println("cwb pos", cwb.Geom.Pos, cwb.Geom.RelPos, e.Pos())
				return true
			})
		}
		mpfr.CursorPos = e.Pos()
		mpfr.startCursor()
	})

	// mpfr.Maker(func(p *tree.Plan) {
	// 	tree.AddAt(p, "mathpadrow", func(w *MathPadRow) {
	// 		txt := NewTextField(w).SetText("")
	// 		txt.OnFocus(func(e events.Event) {
	// 			pa := w.Parent
	// 			pa.(*MathPadFrame).CursorPos = txt.Geom.RelPos.ToPoint().Add(w.Geom.RelPos.ToPoint())
	// 			pa.(*MathPadFrame).startCursor()
	// 		})
	// 		txt.OnKeyChord(func(e events.Event) {
	// 			if keymap.Of(e.KeyChord()) == keymap.Enter {
	// 				row := tree.New[MathPadRow](mpfr)
	// 				fmt.Println("textfield enter pressed")
	// 				txt := core.NewTextField(row).SetText("enter new line")
	// 				txt.OnFocus(func(e events.Event) {
	// 					pa := row.Parent
	// 					pa.(*MathPadFrame).CursorPos = txt.Geom.RelPos.ToPoint().Add(row.Geom.RelPos.ToPoint())
	// 					pa.(*MathPadFrame).startCursor()
	// 				})
	// 				mpfr.AsTree().This.(*MathPadFrame).Update()
	// 				e.SetHandled()
	// 				txt.Send(events.ContextMenu)
	// 			}
	// 		})
	// 		txt1 := NewTextField(w).SetText("")
	// 		txt1.OnFocus(func(e events.Event) {
	// 			pa := w.Parent
	// 			pa.(*MathPadFrame).CursorPos = txt.Geom.RelPos.ToPoint().Add(w.Geom.RelPos.ToPoint())
	// 			pa.(*MathPadFrame).startCursor()
	// 		})
	// 		txt1.OnKeyChord(func(e events.Event) {
	// 			if keymap.Of(e.KeyChord()) == keymap.Enter {
	// 				row := tree.New[MathPadRow](mpfr)
	// 				fmt.Println("textfield enter pressed")
	// 				txt := core.NewTextField(row).SetText("enter new line")
	// 				txt.OnFocus(func(e events.Event) {
	// 					pa := row.Parent
	// 					pa.(*MathPadFrame).CursorPos = txt.Geom.RelPos.ToPoint().Add(row.Geom.RelPos.ToPoint())
	// 					pa.(*MathPadFrame).startCursor()
	// 				})
	// 				mpfr.AsTree().This.(*MathPadFrame).Update()
	// 				e.SetHandled()
	// 				txt.Send(events.ContextMenu)
	// 			}
	// 		})
	// 	})
	// })

}

func (mpfr *MathPadFrame) SetSceneData(sents *Sentences) {
	mpfr.Scene.Data = sents
	for _, sen := range sents.sents {
		NewMathPadRow(mpfr, sen)
	}
}

func (mpfr *MathPadFrame) Render() {
	mpfr.WidgetBase.Render()

	sz := mpfr.Geom.Size.Actual.Content
	mpfr.painter = &mpfr.Scene.Painter
	sty := styles.NewPaint()
	sty.Transform = math32.Translate2D(mpfr.Geom.Pos.Content.X, mpfr.Geom.Pos.Content.Y).Scale(sz.X, sz.Y)
	mpfr.painter.PushContext(sty, nil)
	//mpfr.painter.VectorEffect = ppath.VectorEffectNonScalingStroke
	//mpfr.Draw(mpfr.painter)

	mpfr.painter.Fill.Color = colors.Uniform(color.Red)
	//mpfr.painter.Fill.Opacity = 1
	mpfr.painter.Stroke.Color = colors.Uniform(color.Blue)
	mpfr.painter.Circle(0.5, 0.5, 0.35)
	mpfr.painter.Draw()
	mpfr.Scene.Stage.Main.Sprites.Lock()
	defer mpfr.Scene.Stage.Main.Sprites.Unlock()
	if sp, ok := mpfr.Scene.Stage.Main.Sprites.SpriteByNameNoLock(MathPadFrameSpriteName); ok {
		sp.EventBBox.Min = mpfr.CursorPos
	}

	mpfr.painter.PopContext()

}

// startCursor starts the cursor blinking and renders it
func (mpfr *MathPadFrame) startCursor() {
	fmt.Println("mpfr Cursor Pos", mpfr.CursorPos)
	if mpfr == nil || mpfr.This == nil || !mpfr.IsVisible() {
		return
	}
	if mpfr.IsReadOnly() || !mpfr.AbilityIs(abilities.Focusable) {
		return
	}
	mpfr.toggleCursor(true)
}

// stopCursor stops the cursor from blinking
func (mpfr *MathPadFrame) stopCursor() {
	mpfr.toggleCursor(false)
}

var MathPadFrameLastW tree.Node

// toggleSprite turns on or off the cursor sprite.
func (mpfr *MathPadFrame) toggleCursor(on bool) {
	fmt.Println("start cursor on", on, units.Dp(1).Dots, units.Dp(20).Dots)
	core.TextCursor(on, mpfr.AsWidget(), &MathPadFrameLastW, MathPadFrameSpriteName, 1.01, 23, mpfr.CursorColor, func() image.Point {
		return mpfr.CursorPos //image.Point{100, 200}
	})
}

type MathPadRow struct {
	core.Frame

	painter *paint.Painter
}

var _ = types.AddType(&types.Type{Name: "github.com/runsys/wa.MathPadRow", IDName: "mathpadrow", Doc: "mathpad divide widgets into logical groups and give users the ability\nto math execute.", Embeds: []types.Field{{Name: "MathPadFrame"}}, Fields: []types.Field{{Name: "Type", Doc: "Type is the styling type of the tabs. If it is changed after\nthe tabs are first configured, Update needs to be called on\nthe tabs."}}})

func NewMathPadRow(parent tree.Node, sen *Sentence) *MathPadRow {
	row := tree.New[MathPadRow](parent)
	for sei, se := range sen.children {
		if sei%2 == 0 {
			txt := core.NewText(row).SetText(se.text)
			txt.OnFocus(func(e events.Event) {
				pa := row.Parent
				pa.(*MathPadFrame).CursorPos = txt.Geom.RelPos.ToPoint().Add(row.Geom.RelPos.ToPoint())
				pa.(*MathPadFrame).startCursor()
			})
		} else {
			txt := core.NewTextField(row).SetText(se.text)
			txt.OnFocus(func(e events.Event) {
				pa := row.Parent
				pa.(*MathPadFrame).CursorPos = txt.Geom.RelPos.ToPoint().Add(row.Geom.RelPos.ToPoint())
				pa.(*MathPadFrame).startCursor()
			})
			txt.OnKeyChord(func(e events.Event) {
				if keymap.Of(e.KeyChord()) == keymap.Enter {
					row := tree.New[MathPadRow](parent)
					fmt.Println("textfield enter pressed")
					txt := core.NewTextField(row).SetText("enter new line")
					txt.OnFocus(func(e events.Event) {
						pa := row.Parent
						pa.(*MathPadFrame).CursorPos = txt.Geom.RelPos.ToPoint().Add(row.Geom.RelPos.ToPoint())
						pa.(*MathPadFrame).startCursor()
					})
					parent.AsTree().This.(*MathPadFrame).Update()
					e.SetHandled()
					txt.Send(events.ContextMenu)
				}
			})
		}
	}
	return row
}

func (mpr *MathPadRow) Init() {
	mpr.Frame.Init()
	mpr.Styler(func(s *styles.Style) {
		s.Overflow.Set(styles.OverflowHidden) // no scrollbars!
		s.Gap.Set(units.Dp(4))
		s.Align.Items = styles.Center
		s.Direction = styles.Row
		s.Grow.Set(1, 0)
		s.Wrap = true
	})
}

func (mpr *MathPadRow) Render() {
	mpr.WidgetBase.Render()

	sz := mpr.Geom.Size.Actual.Content
	mpr.painter = &mpr.Scene.Painter
	sty := styles.NewPaint()
	sty.Transform = math32.Translate2D(mpr.Geom.Pos.Content.X, mpr.Geom.Pos.Content.Y).Scale(sz.X, sz.Y)
	mpr.painter.PushContext(sty, nil)
	mpr.painter.VectorEffect = ppath.VectorEffectNonScalingStroke
	//mpr.Draw(mpr.painter)

	//mpr.painter.Fill.Color = colors.Uniform(color.Cyan)
	//mpr.painter.Fill.Opacity = 1
	mpr.painter.Stroke.Color = colors.Uniform(color.Cyan)
	mpr.painter.Stroke.Width = units.Dp(1)
	mpr.painter.Line(1, 0, 1, 1)
	mpr.painter.Draw()

	mpr.painter.PopContext()

}

type SentenceElement struct {
	text string
}

type Sentence struct {
	children []*SentenceElement
}

type Sentences struct {
	sents []*Sentence
}

func main() {
	mainwin := core.NewBody("Mathpad")
	mathpad := core.NewMathpad(mainwin)
	mathpad.SetTooltip("mathpad")
	mainwin.RunMainWindow()
}
