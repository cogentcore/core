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
)

func containsInt(haystack []glfw.Key, needle glfw.Key) bool {
	for _, v := range haystack {
		if needle == v {
			return true
		}
	}
	return false
}

var blankLetterCodes = []glfw.Key{
	glfw.KeyNumLock,
	glfw.KeyDelete,
	glfw.KeyHome,
	glfw.KeyEnd,
	glfw.KeyF1,
	glfw.KeyF2,
	glfw.KeyF3,
	glfw.KeyF4,
	glfw.KeyF5,
	glfw.KeyF6,
	glfw.KeyF7,
	glfw.KeyF8,
	glfw.KeyF9,
	glfw.KeyF10,
	glfw.KeyF11,
	glfw.KeyF12,
	glfw.KeyF13,
	glfw.KeyF14,
	glfw.KeyF15,
	glfw.KeyPageDown,
	glfw.KeyPageUp,
	glfw.KeyLeft,
	glfw.KeyRight,
	glfw.KeyDown,
	glfw.KeyUp,
	glfw.KeyLeftAlt,
	glfw.KeyLeftSuper,
	glfw.KeyLeftControl,
	glfw.KeyLeftShift,
	glfw.KeyRightAlt,
	glfw.KeyRightSuper,
	glfw.KeyRightControl,
	glfw.KeyRightShift,
	glfw.KeyInsert,
	glfw.KeyCapsLock,
}

var keyMapping = map[glfw.Key]string{
	glfw.KeyA:            gi.KeyA,
	glfw.KeyB:            gi.KeyB,
	glfw.KeyC:            gi.KeyC,
	glfw.KeyD:            gi.KeyD,
	glfw.KeyE:            gi.KeyE,
	glfw.KeyF:            gi.KeyF,
	glfw.KeyG:            gi.KeyG,
	glfw.KeyH:            gi.KeyH,
	glfw.KeyI:            gi.KeyI,
	glfw.KeyJ:            gi.KeyJ,
	glfw.KeyK:            gi.KeyK,
	glfw.KeyL:            gi.KeyL,
	glfw.KeyM:            gi.KeyM,
	glfw.KeyN:            gi.KeyN,
	glfw.KeyO:            gi.KeyO,
	glfw.KeyP:            gi.KeyP,
	glfw.KeyQ:            gi.KeyQ,
	glfw.KeyR:            gi.KeyR,
	glfw.KeyS:            gi.KeyS,
	glfw.KeyT:            gi.KeyT,
	glfw.KeyU:            gi.KeyU,
	glfw.KeyV:            gi.KeyV,
	glfw.KeyW:            gi.KeyW,
	glfw.KeyX:            gi.KeyX,
	glfw.KeyY:            gi.KeyY,
	glfw.KeyZ:            gi.KeyZ,
	glfw.Key1:            gi.Key1,
	glfw.Key2:            gi.Key2,
	glfw.Key3:            gi.Key3,
	glfw.Key4:            gi.Key4,
	glfw.Key5:            gi.Key5,
	glfw.Key6:            gi.Key6,
	glfw.Key7:            gi.Key7,
	glfw.Key8:            gi.Key8,
	glfw.Key9:            gi.Key9,
	glfw.Key0:            gi.Key0,
	glfw.KeyGraveAccent:  gi.KeyBackTick,
	glfw.KeyMinus:        gi.KeyMinus,
	glfw.KeyEqual:        gi.KeyEqual,
	glfw.KeyLeftBracket:  gi.KeyLeftBracket,
	glfw.KeyRightBracket: gi.KeyRightBracket,
	glfw.KeyBackslash:    gi.KeyBackslash,
	glfw.KeySemicolon:    gi.KeySemicolon,
	glfw.KeyApostrophe:   gi.KeyQuote,
	glfw.KeyComma:        gi.KeyComma,
	glfw.KeyPeriod:       gi.KeyPeriod,
	glfw.KeySlash:        gi.KeySlash,
	glfw.KeyEnter:        gi.KeyReturn,
	glfw.KeyEscape:       gi.KeyEscape,
	glfw.KeyBackspace:    gi.KeyBackspace,
	glfw.KeyNumLock:      gi.KeyNumlock,
	glfw.KeyDelete:       gi.KeyDelete,
	glfw.KeyHome:         gi.KeyHome,
	glfw.KeyEnd:          gi.KeyEnd,
	glfw.KeyPageUp:       gi.KeyPrior,
	glfw.KeyPageDown:     gi.KeyNext,
	glfw.KeyF1:           gi.KeyF1,
	glfw.KeyF2:           gi.KeyF2,
	glfw.KeyF3:           gi.KeyF3,
	glfw.KeyF4:           gi.KeyF4,
	glfw.KeyF5:           gi.KeyF5,
	glfw.KeyF6:           gi.KeyF6,
	glfw.KeyF7:           gi.KeyF7,
	glfw.KeyF8:           gi.KeyF8,
	glfw.KeyF9:           gi.KeyF9,
	glfw.KeyF10:          gi.KeyF10,
	glfw.KeyF11:          gi.KeyF11,
	glfw.KeyF12:          gi.KeyF12,
	glfw.KeyF13:          gi.KeyF13,
	glfw.KeyF14:          gi.KeyF14,
	glfw.KeyF15:          gi.KeyF15,
	glfw.KeyLeft:         gi.KeyLeftArrow,
	glfw.KeyRight:        gi.KeyRightArrow,
	glfw.KeyDown:         gi.KeyDownArrow,
	glfw.KeyUp:           gi.KeyUpArrow,
	//glfw.KeyFunction:  gi.KeyFunction,
	glfw.KeyLeftAlt:      gi.KeyLeftAlt,
	glfw.KeyRightAlt:     gi.KeyRightAlt,
	glfw.KeyLeftSuper:    gi.KeyLeftSuper,
	glfw.KeyRightSuper:   gi.KeyRightSuper,
	glfw.KeyLeftControl:  gi.KeyLeftControl,
	glfw.KeyRightControl: gi.KeyRightControl,
	glfw.KeyLeftShift:    gi.KeyLeftShift,
	glfw.KeyRightShift:   gi.KeyRightShift,
	glfw.KeyInsert:       gi.KeyInsert,
	glfw.KeyTab:          gi.KeyTab,
	glfw.KeySpace:        gi.KeySpace,
	glfw.KeyKp1:          gi.KeyPadEnd,
	glfw.KeyKp2:          gi.KeyPadDown,
	glfw.KeyKp3:          gi.KeyPadNext,
	glfw.KeyKp4:          gi.KeyPadLeft,
	glfw.KeyKp5:          gi.KeyPadNext,
	glfw.KeyKp6:          gi.KeyPadRight,
	glfw.KeyKp7:          gi.KeyPadHome,
	glfw.KeyKp8:          gi.KeyPadUp,
	glfw.KeyKp9:          gi.KeyPadBegin,
	glfw.KeyKp0:          gi.KeyPadInsert,
	glfw.KeyKpDivide:     gi.KeyPadSlash,
	glfw.KeyKpMultiply:   gi.KeyPadStar,
	glfw.KeyKpSubtract:   gi.KeyPadMinus,
	glfw.KeyKpAdd:        gi.KeyPadPlus,
	glfw.KeyKpDecimal:    gi.KeyPadDot,
	glfw.KeyCapsLock:     gi.KeyCapsLock,
}

func constructChord(key glfw.Key, mods glfw.ModifierKey) (chord string) {
	keys := make(map[string]bool)

	if mods&glfw.ModSuper != 0 {
		keys[gi.KeyLeftSuper] = true
	}

	if mods&glfw.ModShift != 0 {
		keys[gi.KeyLeftShift] = true
	}

	if mods&glfw.ModAlt != 0 {
		keys[gi.KeyLeftAlt] = true
	}

	if mods&glfw.ModControl != 0 {
		keys[gi.KeyLeftControl] = true
	}

	keys[keyMapping[key]] = true

	return gi.ConstructChord(keys)
}
