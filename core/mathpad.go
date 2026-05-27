package core

import (
	"fmt"
	"image"
	"image/color"
	"reflect"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/types"
)

// a Mathematia notepad similar widget.
type Mathpad struct {
	Frame
	toolbar  *Frame
	sentsfrm *MathpadFrame
}

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/core.Mathpad", IDName: "Mathpad", Doc: "Mathpad divide widgets into logical groups and give users the ability\nto math execute.", Embeds: []types.Field{{Name: "MathpadFrame"}}, Fields: []types.Field{{Name: "Type", Doc: "Type is the styling type of the tabs. If it is changed after\nthe tabs are first configured, Update needs to be called on\nthe tabs."}, {Name: "toolbar"}, {Name: "sentsfrm"}}})

// NewMathpad returns a new [Mathpad] with the given optional parent:
// Tabs divide widgets into logical groups and give users the ability
// to freely math execute.
func NewMathpad(parent ...tree.Node) *Mathpad { return tree.New[Mathpad](parent...) }

func (mp *Mathpad) Init() {
	mp.Frame.Init()
	mp.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Direction = styles.Column
	})

	mp.Maker(func(p *tree.Plan) {
		tree.AddAt(p, "frame", func(w *Frame) {
			mp.toolbar = w
			runallbtn := NewButton(w).SetIcon(icons.Add).SetText("Run all")
			runallbtn.OnClick(func(e events.Event) {

			})
			runlinebtn := NewButton(w).SetIcon(icons.Add).SetText("Run line")
			runlinebtn.OnClick(func(e events.Event) {

			})
		})

		tree.AddAt(p, "Mathpadframe", func(w *MathpadFrame) {
			mp.sentsfrm = w
			w.Styler(func(s *styles.Style) {
				s.Overflow.Set(styles.OverflowScroll) // have scrollbar
				s.Grow.Set(1, 1)
				s.Direction = styles.Column
			})
			NewMathpadRow(w, nil, "")
		})
	})
}

type MathpadFrame struct {
	Frame

	painter *paint.Painter

	CursorColor image.Image
	CursorPos   image.Point

	selectInitRowMain                 *MathpadRow
	selectInitPos                     image.Point
	selectInitRow, selectInitRowChild tree.Node
}

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/core.MathpadFrame", IDName: "Mathpadframe", Doc: "Mathpad divide widgets into logical groups and give users the ability\nto math execute.", Embeds: []types.Field{{Name: "MathpadFrame"}}, Fields: []types.Field{{Name: "Type", Doc: "Type is the styling type of the tabs. If it is changed after\nthe tabs are first configured, Update needs to be called on\nthe tabs."}}})

const (
	MathpadFrameSpriteName = " MathpadFrameSpriteName"
)

func (mpfr *MathpadFrame) Init() {
	mpfr.Frame.Init()
	mpfr.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Selectable, abilities.Activatable, abilities.Focusable, abilities.Hoverable, abilities.Draggable, abilities.Clickable, abilities.DoubleClickable, abilities.TripleClickable, abilities.ScrollableUnattended, abilities.Scrollable)
		//s.GrowWrap = false
		s.Grow.Set(1, 1)
		s.Direction = styles.Column
		s.Grow.X = 1
		s.Grow.Y = 1
		s.Gap.Zero()
		s.Align.Content = styles.Start
		s.Align.Items = styles.Start
		s.Overflow.Set(styles.OverflowAuto)
		mpfr.CursorColor = colors.Scheme.Primary.Base
	})
	mpfr.OnClick(func(e events.Event) {
		fmt.Println("Mathpadframe clicked")
		if !mpfr.IsReadOnly() {
			mpfr.SetFocus()
		}
		switch e.MouseButton() {
		case events.Left:
			mpfr.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
				fmt.Println("cwb pos", cwb.Geom.Pos, cwb.Geom.RelPos, e.Pos())
				// if cwb.Geom.Pos.Content.Y-mpfr.Geom.Scroll.Y>=0 && cwb.Geom.Pos.Content.Y-mpfr.Geom.Scroll.Y<mpfr.SceneSize().Y {
				// 	if cwb.Geom.Pos.Content.ToPoint().Y-mpfr.Geom.Scroll.ToPoint().Y>=e.Pos().Y && e.Pos().Y<cwb.Geom.Pos.Content.ToPoint().Y-mpfr.Geom.Scroll.ToPoint().Y+cwb.Geom.Size.Internal.Y {
				// 		cwb.ForWidgetChildren(func(i int, cw1 Widget, cwb1 *WidgetBase) bool {
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
	mpfr.On(events.MouseDown, func(e events.Event) {
		mpfr.selectInitPos = e.Pos()
		mpfr.selectInitRowMain, mpfr.selectInitRow, mpfr.selectInitRowChild = mpfr.pixelToRow(e.Pos())
		fmt.Println("Mathpad mousedown", e.Pos(), "mpfr.selectInitRowMain", mpfr.selectInitRowMain, "mpfr.selectInitRow", mpfr.selectInitRow, "mpfr.selectInitRowChild", mpfr.selectInitRowChild)
	})
	mpfr.On(events.MouseUp, func(e events.Event) {
		selectUpRowMain, selectUpRow, selectUpRowChild := mpfr.pixelToRow(e.Pos())
		rowcmprl := mpfr.compareRow(mpfr.selectInitRowMain, selectUpRowMain)
		fmt.Println("Mathpad mouseup", e.Pos(), "rowcmprl", rowcmprl, "selectUpRowMain", selectUpRow, "selectUpRow", selectUpRow, "selectUpRowChild", selectUpRowChild)
		if rowcmprl == 0 {

		} else if rowcmprl < 0 {
			for childi, child := range mpfr.Children {
				if child == mpfr.selectInitRowMain {
					child.(*MathpadRow).selectToEndByPos(mpfr.selectInitPos, false)
					for i := childi + 1; i < len(mpfr.Children); i += 1 {
						if mpfr.Children[i] != selectUpRowMain {
							mpfr.Children[i].(*MathpadRow).selectAll()
						} else {
							break
						}
					}
					selectUpRowMain.selectToStartByPos(e.Pos(), false)
					break
				}
			}
		} else if rowcmprl > 0 {
			for childi, child := range mpfr.Children {
				if child == selectUpRowMain {
					selectUpRowMain.selectToEndByPos(e.Pos(), false)
					for i := childi + 1; i < len(mpfr.Children); i += 1 {
						if mpfr.Children[i] != mpfr.selectInitRowMain {
							mpfr.Children[i].(*MathpadRow).selectAll()
						} else {
							break
						}
					}
					mpfr.selectInitRowMain.selectToStartByPos(mpfr.selectInitPos, false)
					break
				}
			}
		}
	})
}

func (mpfr *MathpadFrame) pixelToRow(pos image.Point) (out *MathpadRow, row, rowchild tree.Node) {
	for _, child := range mpfr.Children {
		if pos.In(child.(*MathpadRow).Geom.totalRect()) {
			fmt.Println("MathpadFrame pos to row pos", pos, "to", child.(*MathpadRow).PointToRelPos(pos))
			row, rowhild := child.(*MathpadRow).pixelToWidget(pos)
			return child.(*MathpadRow), row, rowhild
		}
	}
	return nil, nil, nil
}

func (mp *MathpadFrame) compareRow(row1, row2 *MathpadRow) (out int) {
	if row1 == row2 {
		return 0
	}
	var row1i, row2i int = -1, -1
	for childi, child := range mp.Children {
		if child == row1 {
			row1i = childi
		}
		if child == row2 {
			row2i = childi
		}
	}
	return row1i - row2i
}

// startCursor starts the cursor blinking and renders it
func (mpfr *MathpadFrame) startCursor() {
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
func (mpfr *MathpadFrame) stopCursor() {
	mpfr.toggleCursor(false)
}

var MathpadFrameLastW tree.Node

// toggleSprite turns on or off the cursor sprite.
func (mpfr *MathpadFrame) toggleCursor(on bool) {
	fmt.Println("start cursor on", on, units.Dp(1).Dots, units.Dp(20).Dots)
	TextCursor(on, mpfr.AsWidget(), &MathpadFrameLastW, MathpadFrameSpriteName, 1.01, 23, mpfr.CursorColor, func() image.Point {
		return mpfr.CursorPos //image.Point{100, 200}
	})
}

type MathpadRow struct {
	Frame

	painter *paint.Painter
}

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/core.MathpadRow", IDName: "Mathpadrow", Doc: "Mathpad divide widgets into logical groups and give users the ability\nto math execute.", Embeds: []types.Field{{Name: "MathpadFrame"}}, Fields: []types.Field{{Name: "Type", Doc: "Type is the styling type of the tabs. If it is changed after\nthe tabs are first configured, Update needs to be called on\nthe tabs."}}})

func NewMathpadRow(parent, after tree.Node, text string) *MathpadRow {
	var row *MathpadRow
	if after == nil {
		row = tree.New[MathpadRow](parent)
	} else {
		row = tree.NewAtAfter[MathpadRow](parent, after)
	}
	row1 := NewFrameRow(row)
	NewText(row1).SetText(" in:")
	inedit := NewMathpadTextField(row1).SetText(text)
	inedit.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 0)
		s.Max.X.Ch(-1)
	})
	inedit.OnKeyChord(func(e events.Event) {
		if keymap.Of(e.KeyChord()) == keymap.Enter {
			text := inedit.Text()
			inedit.SetText(string([]rune(text)[:inedit.cursorPos]))
			newlinetext := string([]rune(text)[inedit.cursorPos:])
			NewMathpadRow(parent, row, newlinetext)
			parent.(*MathpadFrame).Update()
			inedit.SetFocus()
		} else if keymap.Of(e.KeyChord()) == keymap.Backspace {
			if inedit.cursorPos == 0 {
				text := inedit.Text()
				if text == "" {
					for childi, child := range parent.AsTree().Children {
						if child == row {
							parent.AsTree().Children = append(parent.AsTree().Children[:childi], parent.AsTree().Children[childi+1:]...)
							if childi > 0 {
								ed := parent.AsTree().Children[childi-1].AsTree().Children[0].AsTree().Children[1].(*TextField)
								ed.cursorPos = len([]rune(ed.Text()))
								ed.SetText(ed.Text() + text)
								ed.SetFocus()
							}
						}
					}
					parent.(*MathpadFrame).Update()
				} else {
					for childi, child := range parent.AsTree().Children {
						if child == row {
							if childi > 0 {
								ed := parent.AsTree().Children[childi-1].AsTree().Children[0].AsTree().Children[1].(*TextField)
								ed.cursorPos = len([]rune(ed.Text()))
								ed.SetText(ed.Text() + text)
								ed.SetFocus()
								parent.AsTree().Children = append(parent.AsTree().Children[:childi], parent.AsTree().Children[childi+1:]...)
							}
						}
					}
					parent.(*MathpadFrame).Update()
				}
				e.SetHandled()
			}
		}
	})
	inedit.cursorPos = 0
	inedit.SetFocus()
	row2 := NewFrameRow(row)
	NewText(row2).SetText("out:")
	outtext := NewText(row2).SetText("")
	outtext.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 0)
	})
	row2.SetState(true, states.Invisible)
	return row
}

func (mpr *MathpadRow) Init() {
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

func (mpr *MathpadRow) pixelToWidget(pos image.Point) (row, rowchild tree.Node) {
	for _, child := range mpr.Children {
		fmt.Println("MathpadRow pos", pos, "childrect", child.(*Frame).Geom.totalRect())
		if pos.In(child.(*Frame).Geom.totalRect()) {
			for _, child2 := range child.(*Frame).Children {
				switch child2.(type) {
				case *Text:
					if pos.In(child2.(*Text).Geom.totalRect()) {
						return child, child2
					}
				case *MathpadTextField:
					if pos.In(child2.(*MathpadTextField).Geom.totalRect()) {
						return child, child2
					}
				default:
					panic("error type:" + reflect.TypeOf(child2).String())
				}
			}
		}
	}
	return nil, nil
}

func (mpr *MathpadRow) selectAll() {
	mpr.Children[0].(*Frame).Children[1].(*MathpadTextField).selectStartToEnd()
}

func (mpr *MathpadRow) selectToEndByPos(pos image.Point, cursorAtLeft bool) {
	edit := mpr.Children[0].(*Frame).Children[1].(*MathpadTextField)
	edit.selectToEndByPos(pos, cursorAtLeft)
}
func (mpr *MathpadRow) selectToStartByPos(pos image.Point, cursorAtLeft bool) {
	edit := mpr.Children[0].(*Frame).Children[1].(*MathpadTextField)
	edit.selectToStartByPos(pos, cursorAtLeft)
}

func (mpr *MathpadRow) Render() {
	mpr.WidgetBase.Render()
	sz := mpr.Geom.Size.Actual.Content
	mpr.painter = &mpr.Scene.Painter
	sty := styles.NewPaint()
	sty.Transform = math32.Translate2D(mpr.Geom.Pos.Content.X, mpr.Geom.Pos.Content.Y).Scale(sz.X, sz.Y)
	mpr.painter.PushContext(sty, nil)
	mpr.painter.VectorEffect = ppath.VectorEffectNonScalingStroke
	mpr.painter.Stroke.Color = colors.Uniform(color.White)
	mpr.painter.Stroke.Width = units.Dp(1)
	mpr.painter.Line(1, 0, 1, 1)
	mpr.painter.Draw()
	mpr.painter.PopContext()
}
