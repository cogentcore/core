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

package cocoa

import "github.com/rcoreilly/goki/gi"

func containsInt(haystack []int, needle int) bool {
	for _, v := range haystack {
		if needle == v {
			return true
		}
	}
	return false
}

var blankLetterCodes = []int{71, 117, 115, 119, 116, 121, 122, 120, 99, 118, 96, 97, 98, 100, 101, 109, 10, 103, 111, 105, 107, 113, 123, 124, 125, 126, 63, 58, 55, 59, 56, 61, 54, 62, 60, 114}
var keyMapping = map[int]string{
	0:   gi.KeyA,
	11:  gi.KeyB,
	8:   gi.KeyC,
	2:   gi.KeyD,
	14:  gi.KeyE,
	3:   gi.KeyF,
	5:   gi.KeyG,
	4:   gi.KeyH,
	34:  gi.KeyI,
	38:  gi.KeyJ,
	40:  gi.KeyK,
	37:  gi.KeyL,
	46:  gi.KeyM,
	45:  gi.KeyN,
	31:  gi.KeyO,
	35:  gi.KeyP,
	12:  gi.KeyQ,
	15:  gi.KeyR,
	1:   gi.KeyS,
	17:  gi.KeyT,
	32:  gi.KeyU,
	9:   gi.KeyV,
	13:  gi.KeyW,
	7:   gi.KeyX,
	16:  gi.KeyY,
	6:   gi.KeyZ,
	18:  gi.Key1,
	19:  gi.Key2,
	20:  gi.Key3,
	21:  gi.Key4,
	23:  gi.Key5,
	22:  gi.Key6,
	26:  gi.Key7,
	28:  gi.Key8,
	25:  gi.Key9,
	29:  gi.Key0,
	50:  gi.KeyBackTick,
	27:  gi.KeyMinus,
	24:  gi.KeyEqual,
	33:  gi.KeyLeftBracket,
	30:  gi.KeyRightBracket,
	42:  gi.KeyBackslash,
	41:  gi.KeySemicolon,
	39:  gi.KeyQuote,
	43:  gi.KeyComma,
	47:  gi.KeyPeriod,
	44:  gi.KeySlash,
	36:  gi.KeyReturn,
	53:  gi.KeyEscape,
	51:  gi.KeyBackspace,
	71:  gi.KeyNumlock,
	117: gi.KeyDelete,
	115: gi.KeyHome,
	119: gi.KeyEnd,
	116: gi.KeyPrior,
	121: gi.KeyNext,
	122: gi.KeyF1,
	120: gi.KeyF2,
	99:  gi.KeyF3,
	118: gi.KeyF4,
	96:  gi.KeyF5,
	97:  gi.KeyF6,
	98:  gi.KeyF7,
	100: gi.KeyF8,
	101: gi.KeyF9,
	109: gi.KeyF10,
	103: gi.KeyF11,
	111: gi.KeyF12,
	105: gi.KeyF13,
	107: gi.KeyF14,
	113: gi.KeyF15,
	123: gi.KeyLeftArrow,
	124: gi.KeyRightArrow,
	125: gi.KeyDownArrow,
	126: gi.KeyUpArrow,
	63:  gi.KeyFunction,
	58:  gi.KeyLeftAlt,
	61:  gi.KeyRightAlt,
	55:  gi.KeyLeftSuper,
	54:  gi.KeyRightSuper,
	59:  gi.KeyLeftControl,
	62:  gi.KeyRightControl,
	56:  gi.KeyLeftShift,
	60:  gi.KeyRightShift,
	114: gi.KeyInsert,
	48:  gi.KeyTab,
	49:  gi.KeySpace,
	83:  gi.KeyPadHome, // keypad
	84:  gi.KeyPadDown,
	85:  gi.KeyPadNext,
	86:  gi.KeyPadLeft,
	87:  gi.KeyPadBegin,
	88:  gi.KeyPadRight,
	89:  gi.KeyPadEnd,
	91:  gi.KeyPadUp,
	92:  gi.KeyPadNext,
	82:  gi.KeyPadInsert,
	75:  gi.KeyPadSlash,
	67:  gi.KeyPadStar,
	78:  gi.KeyPadMinus,
	69:  gi.KeyPadPlus,
	65:  gi.KeyPadDot,
}
