// Copyright 2023 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ios

package ios

/*
#cgo CFLAGS: -x objective-c -DGL_SILENCE_DEPRECATION
#cgo LDFLAGS: -framework Foundation -framework UIKit -framework MobileCoreServices -framework QuartzCore -framework UserNotifications
#include <sys/utsname.h>
#include <stdint.h>
#include <stdbool.h>
#include <pthread.h>
#import <UIKit/UIKit.h>
#import <MobileCoreServices/MobileCoreServices.h>
#include <UIKit/UIDevice.h>
*/
import "C"
import (
	"image"

	"goki.dev/goosi/events"
	"goki.dev/goosi/events/key"
)

// touchIDs are the current active touches. The position in the array
// is the ID, the value is the UITouch* pointer value.
//
// It is widely reported that the iPhone can handle up to 5 simultaneous
// touch events, while the iPad can handle 11.
var touchIDs [11]uintptr

//export sendTouch
func sendTouch(cTouch, cTouchType uintptr, x, y float32) {
	id := -1
	for i, val := range touchIDs {
		if val == cTouch {
			id = i
			break
		}
	}
	if id == -1 {
		for i, val := range touchIDs {
			if val == 0 {
				touchIDs[i] = cTouch
				id = i
				break
			}
		}
		if id == -1 {
			panic("out of touchIDs")
		}
	}
	t := events.TouchStart
	switch cTouchType {
	case 0:
		t = events.TouchStart
	case 1:
		t = events.TouchMove
	case 2:
		t = events.TouchEnd
		// Clear all touchIDs when touch ends. The UITouch pointers are unique
		// at every multi-touch event. See:
		// https://github.com/fyne-io/fyne/issues/2407
		// https://developer.apple.com/documentation/uikit/touches_presses_and_gestures?language=objc
		for idx := range touchIDs {
			touchIDs[idx] = 0
		}
	}

	theApp.window.EvMgr.Touch(t, events.Sequence(id), image.Pt(int(x), int(y)))
}

//export keyboardTyped
func keyboardTyped(str *C.char) {
	for _, r := range C.GoString(str) {
		code := getCodeFromRune(r)
		theApp.window.EvMgr.KeyChord(r, code, 0) // TODO: modifiers
	}
}

//export keyboardDelete
func keyboardDelete() {
	theApp.window.EvMgr.KeyChord(0, key.CodeDeleteBackspace, 0) // TODO: modifiers
}

var codeRune = map[rune]key.Codes{
	'0':  key.Code0,
	'1':  key.Code1,
	'2':  key.Code2,
	'3':  key.Code3,
	'4':  key.Code4,
	'5':  key.Code5,
	'6':  key.Code6,
	'7':  key.Code7,
	'8':  key.Code8,
	'9':  key.Code9,
	'a':  key.CodeA,
	'b':  key.CodeB,
	'c':  key.CodeC,
	'd':  key.CodeD,
	'e':  key.CodeE,
	'f':  key.CodeF,
	'g':  key.CodeG,
	'h':  key.CodeH,
	'i':  key.CodeI,
	'j':  key.CodeJ,
	'k':  key.CodeK,
	'l':  key.CodeL,
	'm':  key.CodeM,
	'n':  key.CodeN,
	'o':  key.CodeO,
	'p':  key.CodeP,
	'q':  key.CodeQ,
	'r':  key.CodeR,
	's':  key.CodeS,
	't':  key.CodeT,
	'u':  key.CodeU,
	'v':  key.CodeV,
	'w':  key.CodeW,
	'x':  key.CodeX,
	'y':  key.CodeY,
	'z':  key.CodeZ,
	'A':  key.CodeA,
	'B':  key.CodeB,
	'C':  key.CodeC,
	'D':  key.CodeD,
	'E':  key.CodeE,
	'F':  key.CodeF,
	'G':  key.CodeG,
	'H':  key.CodeH,
	'I':  key.CodeI,
	'J':  key.CodeJ,
	'K':  key.CodeK,
	'L':  key.CodeL,
	'M':  key.CodeM,
	'N':  key.CodeN,
	'O':  key.CodeO,
	'P':  key.CodeP,
	'Q':  key.CodeQ,
	'R':  key.CodeR,
	'S':  key.CodeS,
	'T':  key.CodeT,
	'U':  key.CodeU,
	'V':  key.CodeV,
	'W':  key.CodeW,
	'X':  key.CodeX,
	'Y':  key.CodeY,
	'Z':  key.CodeZ,
	',':  key.CodeComma,
	'.':  key.CodeFullStop,
	' ':  key.CodeSpacebar,
	'\n': key.CodeReturnEnter,
	'`':  key.CodeGraveAccent,
	'-':  key.CodeHyphenMinus,
	'=':  key.CodeEqualSign,
	'[':  key.CodeLeftSquareBracket,
	']':  key.CodeRightSquareBracket,
	'\\': key.CodeBackslash,
	';':  key.CodeSemicolon,
	'\'': key.CodeApostrophe,
	'/':  key.CodeSlash,
}

func getCodeFromRune(r rune) key.Codes {
	if code, ok := codeRune[r]; ok {
		return code
	}
	return key.CodeUnknown
}
