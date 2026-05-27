package core

import (
	"fmt"
	"image"
	"image/color"

	//"reflect"
	"unicode"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/ppath"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"

	//"cogentcore.org/core/styles/states"
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
			runallbtn := NewButton(w).SetText("Run all")
			runallbtn.OnClick(func(e events.Event) {

			})
			runlinebtn := NewButton(w).SetText("Run line")
			runlinebtn.OnClick(func(e events.Event) {

			})
		})

		tree.AddAt(p, "Mathpadframe", func(w *MathpadFrame) {
			mp.sentsfrm = w
			NewMathpadRow(w, nil, "", true)
		})
	})
}

type MathpadFrame struct {
	Frame

	painter *paint.Painter

	CursorColor image.Image
	CursorPos   image.Point

	selectInitPos      image.Point
	selectInitRow      *MathpadRow
	selectInitRowChild Widget

	focusChild tree.Node
}

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/core.MathpadFrame", IDName: "Mathpadframe", Doc: "Mathpad divide widgets into logical groups and give users the ability\nto math execute.", Embeds: []types.Field{{Name: "MathpadFrame"}}, Fields: []types.Field{{Name: "Type", Doc: "Type is the styling type of the tabs. If it is changed after\nthe tabs are first configured, Update needs to be called on\nthe tabs."}}})

const (
	MathpadFrameSpriteName = " MathpadFrameSpriteName"
)

func (mpfr *MathpadFrame) Init() {
	mpfr.Frame.Init()
	mpfr.Styler(func(s *styles.Style) {
		s.SetAbilities(true, abilities.Activatable, abilities.Focusable, abilities.Hoverable, abilities.Draggable, abilities.Clickable, abilities.DoubleClickable, abilities.TripleClickable, abilities.ScrollableUnattended, abilities.Scrollable)
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
		s.Overflow.Set(styles.OverflowScroll) // have scrollbar
		s.Grow.Set(1, 1)
		s.Direction = styles.Column
	})

	// mpfr.OnClick(func(e events.Event) {
	// 	fmt.Println("Mathpadframe clicked")
	// 	if !mpfr.IsReadOnly() {
	// 		mpfr.SetFocus()
	// 	}
	// 	switch e.MouseButton() {
	// 	case events.Left:
	// 		mpfr.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
	// 			fmt.Println("cwb pos", cwb.Geom.Pos, cwb.Geom.RelPos, e.Pos())
	// 			// if cwb.Geom.Pos.Content.Y-mpfr.Geom.Scroll.Y>=0 && cwb.Geom.Pos.Content.Y-mpfr.Geom.Scroll.Y<mpfr.SceneSize().Y {
	// 			// 	if cwb.Geom.Pos.Content.ToPoint().Y-mpfr.Geom.Scroll.ToPoint().Y>=e.Pos().Y && e.Pos().Y<cwb.Geom.Pos.Content.ToPoint().Y-mpfr.Geom.Scroll.ToPoint().Y+cwb.Geom.Size.Internal.Y {
	// 			// 		cwb.ForWidgetChildren(func(i int, cw1 Widget, cwb1 *WidgetBase) bool {
	// 			// 			if cwb1.Geom.Pos.Content.X>e.Pos().X && e.Pos().X<cwb1.Geom.Pos.Content.X+cwb1.Geom.Size.Internal.Y {
	// 			// 				mpfr.CursorPos=cwb1.Geom.RelPos.ToPoint().Add(cwb.Geom.RelPos.ToPoint())
	// 			// 				return false
	// 			// 			}
	// 			// 			fmt.Println("cwb pos", cwb.Geom.Pos, cwb.Geom.RelPos, e.Pos())
	// 			// 			return true
	// 			// 		})
	// 			// 		return false
	// 			// 	}
	// 			// }
	// 			// fmt.Println("cwb pos", cwb.Geom.Pos, cwb.Geom.RelPos, e.Pos())
	// 			return true
	// 		})
	// 	}
	// 	mpfr.CursorPos = e.Pos()
	// 	mpfr.startCursor()
	// })
	mpfr.On(events.MouseDown, func(e events.Event) {
		mpfr.selectInitPos = e.Pos()
		mpfr.selectInitRow, mpfr.selectInitRowChild = mpfr.pixelToRow(e.Pos())
		fmt.Println("Mathpad mousedown", e.Pos(), "mpfr.selectInitRow", mpfr.selectInitRow, "mpfr.selectInitRowChild", mpfr.selectInitRowChild)
	})
	mpfr.On(events.MouseUp, func(e events.Event) {
		if e.Pos().Eq(mpfr.selectInitPos) {
			mpfr.focusChild = mpfr.selectInitRowChild
			switch wid := mpfr.focusChild.(type) {
			case *MathpadTextField:
				wid.cursorPos = wid.pixelToCursor(e.Pos())
				wid.startCursor()
			default:
				mpfr.CursorPos = e.Pos()
			}
			fmt.Println("mpfr.SetFocus()")
			mpfr.SetFocus()
		} else {
			Row, RowChild := mpfr.pixelToRow(e.Pos())
			fmt.Println("Row, RowChild", Row, RowChild, "mpfr.selectInitRow, mpfr.selectInitRowChild", mpfr.selectInitRow, mpfr.selectInitRowChild)
			if RowChild == mpfr.selectInitRowChild { //select in row
				fmt.Println("RowChild == mpfr.selectInitRowChild")
			} else {
				rowcmprl := mpfr.compareRow(mpfr.selectInitRow, Row)
				fmt.Println("rowcmprl", rowcmprl)
				if rowcmprl == 0 {

				} else if rowcmprl < 0 {
					for childi := 0; childi < len(mpfr.Children); childi += 1 {
						child := mpfr.Children[childi]
						if child == mpfr.selectInitRow {
							mpfr.selectInitRowChild.(*MathpadTextField).selectToEndByPos(mpfr.selectInitPos, false)
							fmt.Println("cwb.Children[2]")
							for childi = childi + 1; childi < len(mpfr.Children); childi += 1 {
								if mpfr.Children[childi] != Row {
									fmt.Println("cwb.Children[3]", mpfr.Children[childi])
									mpfr.Children[childi].(*MathpadRow).Children[1].(*MathpadTextField).selectAll()
								} else {
									break
								}
							}
							RowChild.(*MathpadTextField).clearSelected()
							RowChild.(*MathpadTextField).selectToStartByPos(e.Pos(), false)
							RowChild.(*MathpadTextField).toggleCursor(true)
						} else {
							child.(*MathpadRow).Children[1].(*MathpadTextField).clearSelected()
						}
					}
				} else if rowcmprl > 0 {
					for childi := 0; childi < len(mpfr.Children); childi += 1 {
						child := mpfr.Children[childi]
						if child == Row {
							RowChild.(*MathpadTextField).clearSelected()
							RowChild.(*MathpadTextField).selectToEndByPos(e.Pos(), false)
							RowChild.(*MathpadTextField).cursorPos = RowChild.(*MathpadTextField).pixelToCursor(e.Pos())
							RowChild.(*MathpadTextField).toggleCursor(true)
							RowChild.(*MathpadTextField).startCursor()
							fmt.Println("cwb.Children[2]")
							for childi = childi + 1; childi < len(mpfr.Children); childi += 1 {
								if mpfr.Children[childi] != mpfr.selectInitRow {
									fmt.Println("cwb.Children[3]", mpfr.Children[childi])
									mpfr.Children[childi].(*MathpadRow).Children[1].(*MathpadTextField).selectAll()
								} else {
									break
								}
							}
							mpfr.selectInitRowChild.(*MathpadTextField).selectToStartByPos(mpfr.selectInitPos, false)
						} else {
							child.(*MathpadRow).Children[1].(*MathpadTextField).clearSelected()
						}
					}
				}
			}
		}
		// if mpfr.selectInitRowMain != nil {
		// 	if e.Pos().Eq(mpfr.selectInitPos) {
		// 		focusChild=
		// 	}
		// 	selectUpRowMain, selectUpRow, selectUpRowChild := mpfr.pixelToRow(e.Pos())
		// 	rowcmprl := mpfr.compareRow(mpfr.selectInitRowMain, selectUpRowMain)
		// 	fmt.Println("Mathpad mouseup", e.Pos(), "rowcmprl", rowcmprl, "selectUpRowMain", selectUpRow, "selectUpRow", selectUpRow, "selectUpRowChild", selectUpRowChild)
		// 	if rowcmprl == 0 {

		// 	} else if rowcmprl < 0 {
		// 		for childi, child := range mpfr.Children {
		// 			if child == mpfr.selectInitRowMain {
		// 				child.(*MathpadRow).selectToEndByPos(mpfr.selectInitPos, false)
		// 				for i := childi + 1; i < len(mpfr.Children); i += 1 {
		// 					if mpfr.Children[i] != selectUpRowMain {
		// 						mpfr.Children[i].(*MathpadRow).selectAll()
		// 					} else {
		// 						break
		// 					}
		// 				}
		// 				selectUpRowMain.selectToStartByPos(e.Pos(), false)
		// 				selectUpRowChild.(*MathpadTextField).toggleCursor(true)
		// 				break
		// 			}
		// 		}
		// 	} else if rowcmprl > 0 {
		// 		for childi, child := range mpfr.Children {
		// 			if child == selectUpRowMain {
		// 				selectUpRowMain.selectToEndByPos(e.Pos(), false)
		// 				for i := childi + 1; i < len(mpfr.Children); i += 1 {
		// 					if mpfr.Children[i] != mpfr.selectInitRowMain {
		// 						mpfr.Children[i].(*MathpadRow).selectAll()
		// 					} else {
		// 						break
		// 					}
		// 				}
		// 				mpfr.selectInitRowMain.selectToStartByPos(mpfr.selectInitPos, false)
		// 				selectUpRowChild.(*MathpadTextField).cursorPos = selectUpRowChild.(*MathpadTextField).pixelToCursor(e.Pos())
		// 				selectUpRowChild.(*MathpadTextField).toggleCursor(true)
		// 				break
		// 			}
		// 		}
		// 	}
		// } else {
		// 	mpfr.CursorPos = e.Pos()
		// 	mpfr.startCursor()
		// }
	})

	mpfr.OnKeyChord(func(e events.Event) {
		kf := keymap.Of(e.KeyChord())
		fmt.Println("mpfr OnKeyChord", kf)
		// first all the keys that work for both inactive and active
		switch kf {
		case keymap.MoveRight:
			//e.SetHandled()
		case keymap.MoveLeft:
			//e.SetHandled()
		case keymap.MoveDown:
		case keymap.MoveUp:
		case keymap.Home:
		case keymap.End:
			//e.SetHandled()
		case keymap.SelectMode:
			//e.SetHandled()
		case keymap.CancelSelect:
			//e.SetHandled()
		case keymap.SelectAll:
			//e.SetHandled()
		case keymap.Copy:
			e.SetHandled()
			mpfr.Clipboard().Write(mimedata.NewText(mpfr.selection()))
		}
		if e.IsHandled() {
			return
		}
		switch kf {
		case keymap.Enter:
			switch wid := mpfr.focusChild.(type) {
			case *MathpadTextField:
				text := wid.Text()
				wid.SetText(string([]rune(text)[:wid.cursorPos]))
				newlinetext := string([]rune(text)[wid.cursorPos:])
				mpfr.selectInitRow, mpfr.selectInitRowChild = NewMathpadRow(mpfr, mpfr.selectInitRow, newlinetext, true)
				mpfr.focusChild = mpfr.selectInitRowChild
				mpfr.selectInitRowChild.(*MathpadTextField).cursorPos = 0
				mpfr.selectInitRowChild.(*MathpadTextField).startCursor()
				mpfr.Update()
				mpfr.SetFocus()
				e.SetHandled()
			default:
			}
		case keymap.FocusNext: // we process tab to make it EditDone as opposed to other ways of losing focus
			//e.SetHandled()
		case keymap.Accept: // ctrl+enter
			//e.SetHandled()
		case keymap.FocusPrev:
			//e.SetHandled()
		case keymap.Abort: // esc
			//e.SetHandled()
		case keymap.Backspace:
		case keymap.Delete:
		case keymap.Cut:
		case keymap.Paste:
		case keymap.Undo:
		case keymap.Redo:
		case keymap.None:
			if unicode.IsPrint(e.KeyRune()) {
				if !e.HasAnyModifier(key.Control, key.Meta) {
					e.SetHandled()
					switch wid := mpfr.focusChild.(type) {
					case *MathpadTextField:
						wid.saveUndo()
						wid.insertAtCursor(string(e.KeyRune()))
						if e.KeyRune() == ' ' {
							wid.cancelComplete()
						} else {
							wid.offerComplete()
						}
						//wid.Send(events.Input, e)
						wid.updateCursorPosition()
						wid.startCursor()
					default:
					}
				}
			}
		}
	})
}

func (mpfr *MathpadFrame) selection() (out string) {
	outbytes := []byte{}
	for _, child := range mpfr.Children {
		outbytes = append(outbytes, []byte(child.(*MathpadRow).Children[0].(*Frame).Children[1].(*MathpadTextField).selection())...)
	}
	return string(outbytes)
}

func (mpfr *MathpadFrame) clearSelect() {
	for _, child := range mpfr.Children {
		child.(*MathpadRow).Children[0].(*Frame).Children[1].(*MathpadTextField).clearSelected()
	}
}

func (mpfr *MathpadFrame) pixelToRow(pos image.Point) (row *MathpadRow, rowchild Widget) {
	mpfr.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		if pos.In(cwb.Geom.totalRect()) {
			rowchild = cw.(*MathpadRow).pixelToWidgetBase(pos)
			row = cw.(*MathpadRow)
			return tree.Break
		}
		return tree.Continue
	})
	return row, rowchild
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

var _ = types.AddType(&types.Type{Name: "cogentcore.org/core/core.MathpadRow", IDName: "mathpadrow", Doc: "Mathpad divide widgets into logical groups and give users the ability\nto math execute.", Embeds: []types.Field{{Name: "MathpadFrame"}}, Fields: []types.Field{{Name: "Type", Doc: "Type is the styling type of the tabs. If it is changed after\nthe tabs are first configured, Update needs to be called on\nthe tabs."}}})

func NewMathpadRow(parent, after tree.Node, text string, inrow bool) (row *MathpadRow, rowchild Widget) {
	if after == nil {
		row = tree.New[MathpadRow](parent)
	} else {
		row = tree.NewAtAfter[MathpadRow](parent, after)
	}
	if inrow {
		NewText(row).SetText(" in:")
	} else {
		NewText(row).SetText("out:")
	}
	inedit := NewMathpadTextField(row).SetText(text)
	inedit.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 0)
		s.Max.X.Ch(-1)
	})
	// inedit.OnKeyChord(func(e events.Event) {
	// 	kf := keymap.Of(e.KeyChord())
	// 	fmt.Println("inedit.OnKeyChord", kf)
	// 	switch kf {
	// 	case keymap.Copy:
	// 		parent.(*MathpadFrame).Send(events.KeyChord, e)
	// 	case keymap.Enter:
	// 		text := inedit.Text()
	// 		inedit.SetText(string([]rune(text)[:inedit.cursorPos]))
	// 		newlinetext := string([]rune(text)[inedit.cursorPos:])
	// 		NewMathpadRow(parent, row, newlinetext, true)
	// 		parent.(*MathpadFrame).Update()
	// 		e.SetHandled()
	// 	case keymap.Backspace:
	// 		if inedit.cursorPos == 0 {
	// 			text := inedit.Text()
	// 			if text == "" {
	// 				for childi, child := range parent.AsTree().Children {
	// 					if child == row {
	// 						parent.AsTree().Children = append(parent.AsTree().Children[:childi], parent.AsTree().Children[childi+1:]...)
	// 						if childi > 0 {
	// 							ed := parent.AsTree().Children[childi-1].AsTree().Children[0].AsTree().Children[1].(*TextField)
	// 							ed.cursorPos = len([]rune(ed.Text()))
	// 							ed.SetText(ed.Text() + text)
	// 							ed.SetFocus()
	// 						}
	// 					}
	// 				}
	// 				parent.(*MathpadFrame).Update()
	// 			} else {
	// 				for childi, child := range parent.AsTree().Children {
	// 					if child == row {
	// 						if childi > 0 {
	// 							ed := parent.AsTree().Children[childi-1].AsTree().Children[0].AsTree().Children[1].(*TextField)
	// 							ed.cursorPos = len([]rune(ed.Text()))
	// 							ed.SetText(ed.Text() + text)
	// 							ed.SetFocus()
	// 							parent.AsTree().Children = append(parent.AsTree().Children[:childi], parent.AsTree().Children[childi+1:]...)
	// 						}
	// 					}
	// 				}
	// 				parent.(*MathpadFrame).Update()
	// 			}
	// 			e.SetHandled()
	// 		}
	// 	default:
	// 		e.ClearHandled()
	// 	}
	// })
	inedit.cursorPos = 0
	inedit.SetFocus()
	return row, inedit
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

func (mpr *MathpadRow) pixelToWidgetBase(pos image.Point) (child Widget) {
	mpr.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		fmt.Println("cwb.Geom.totalRect()", cwb.Geom.TotalBBox)
		if pos.In(cwb.Geom.TotalBBox) {
			child = cw
			return tree.Break
		}
		return tree.Continue
	})
	return child
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
