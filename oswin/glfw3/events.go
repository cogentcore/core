/*
   Copyright 2012 the go.wde authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package glfw3

import (
	glfw "github.com/grd/glfw3"
	"github.com/rcoreilly/goki/gi"
	"image"
	"math"
	"time"
)

var lastCursorPosition image.Point

func getMouseButton(button glfw.MouseButton) gi.MouseButton {
	switch button {
	case glfw.MouseButtonLeft:
		return gi.LeftButton
	case glfw.MouseButtonMiddle:
		return gi.MiddleButton
	case glfw.MouseButtonRight:
		return gi.RightButton
	}
	return 0
}

func mouseButtonCallback(w *glfw.Window, button glfw.MouseButton,
	action glfw.Action, mod glfw.ModifierKey) {

	switch action {

	case glfw.Release:
		var bue gi.MouseUpEvent
		bue.Which = getMouseButton(button)
		x, y := w.GetCursorPosition()
		bue.Where.X = int(math.Floor(x))
		bue.Where.Y = int(math.Floor(y))
		if ws, ok := windowMap[w.C()]; ok {
			ws.events <- bue
		}

	case glfw.Press:
		var bde gi.MouseDownEvent
		bde.Which = getMouseButton(button)
		x, y := w.GetCursorPosition()
		bde.Where.X = int(math.Floor(x))
		bde.Where.Y = int(math.Floor(y))
		if ws, ok := windowMap[w.C()]; ok {
			ws.events <- bde
		}
	}
}

func buttonForDetail(detail glfw.MouseButton) gi.MouseButton {
	switch detail {
	case glfw.MouseButtonLeft:
		return gi.LeftButton
	case glfw.MouseButtonMiddle:
		return gi.MiddleButton
	case glfw.MouseButtonRight:
		return gi.RightButton
		//
		// Mouse wheel button Up and Down are not implemented (yet).
		//
		// case 4:
		//	return gi.WheelUpButton
		// case 5:
		//	return gi.WheelDownButton
	}
	return 0
}

func cursorEnterCallback(w *glfw.Window, entered bool) {
	var event interface{}

	if entered {
		var ene gi.MouseEnteredEvent
		x, y := w.GetCursorPosition()
		ene.Where.X = int(math.Floor(x))
		ene.Where.Y = int(math.Floor(y))
		event = ene
	} else {
		var exe gi.MouseExitedEvent
		x, y := w.GetCursorPosition()
		exe.Where.X = int(math.Floor(x))
		exe.Where.Y = int(math.Floor(y))
		event = exe
	}

	if ws, ok := windowMap[w.C()]; ok {
		ws.events <- event
	}
}

func cursorPositionCallback(w *glfw.Window, xpos float64, ypos float64) {
	cursorPosition := image.Point{int(xpos), int(ypos)}
	if ws, ok := windowMap[w.C()]; ok {
		var event gi.MouseMovedEvent
		event.From = lastCursorPosition
		event.Where = cursorPosition
		ws.events <- event
	}
	lastCursorPosition = cursorPosition
}

func framebufferSizeCallback(w *glfw.Window, width int, height int) {
	event := gi.ResizeEvent{width, height}
	if ws, ok := windowMap[w.C()]; ok {
		ws.buffer.RGBA = image.NewRGBA(image.Rect(0, 0, width, height))
		ws.events <- event
	}
}

func keyCallback(w *glfw.Window, key glfw.Key, scancode int,
	action glfw.Action, mods glfw.ModifierKey) {

	ws := windowMap[w.C()]
	if ws == nil {
		return
	}

	switch action {

	case glfw.Press:
		var letter string
		var ke gi.KeyEvent

		blankLetter := containsInt(blankLetterCodes, key)
		if !blankLetter {
			letter = string(key)
		}

		ke.Key = keyMapping[key]

		ws.events <- gi.KeyDownEvent(ke)

		kte := gi.KeyTypedEvent{
			KeyEvent: ke,
			Chord:    constructChord(key, mods),
			Glyph:    letter,
		}

		keyChan <- keyType{ws, kte}

	case glfw.Repeat:
		var letter string

		blankLetter := containsInt(blankLetterCodes, key)
		if !blankLetter {
			letter = string(key)
		}

		ke := gi.KeyEvent{keyMapping[key]}

		kte := gi.KeyTypedEvent{
			KeyEvent: ke,
			Chord:    constructChord(key, mods),
			Glyph:    letter,
		}

		keyChan <- keyType{ws, kte}

	case glfw.Release:
		var ke gi.KeyUpEvent
		ke.Key = keyMapping[key]
		ws.events <- ke
	}

}

func characterCallback(w *glfw.Window, char rune) {
	ws := windowMap[w.C()]
	if ws == nil {
		return
	}
	characterChan <- characterType{ws, char}
}

type keyType struct {
	window *OSWindow
	ke     gi.KeyTypedEvent
}

type characterType struct {
	window *OSWindow
	char   rune
}

var (
	keyChan       = make(chan keyType)
	characterChan = make(chan characterType)
)

func setGlyph() {
	for {
		key := <-keyChan

		select {
		case newKey := <-keyChan:
			key.window.events <- key.ke
			keyChan <- newKey
		case ch := <-characterChan:
			key.ke.Glyph = string(ch.char)
			key.window.events <- key.ke
		case <-time.After(10 * time.Millisecond):
			key.window.events <- key.ke
		}
	}
}

func (w *OSWindow) EventChan() (events <-chan interface{}) {
	events = w.events
	return
}
