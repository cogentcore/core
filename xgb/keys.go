package xgb

import (
	"github.com/rcoreilly/goki/gi"
)

func keyForCode(code string) (key string) {
	key = codeKeys[code]
	if key == "" {
		key = code
	}
	return
}

func letterForCode(code string) (letter string) {
	if len(code) == 1 {
		letter = code
	} else {
		letter = longLetters[code]
	}
	return
}

var codeKeys map[string]string
var longLetters = map[string]string{
	"quoteleft":  "`",
	"quoteright": "'",
}

func init() {
	codeKeys = map[string]string{
		"Shift_L":          gi.KeyLeftShift,
		"Shift_R":          gi.KeyRightShift,
		"Control_L":        gi.KeyLeftControl,
		"Control_R":        gi.KeyRightControl,
		"Hangul_switch":    gi.KeyLeftAlt,
		"Alt_L":            gi.KeyLeftAlt,
		"Alt_R":            gi.KeyRightAlt,
		"Meta_L":           gi.KeyLeftSuper,
		"Meta_R":           gi.KeyRightSuper,
		"Super_L":          gi.KeyLeftSuper,
		"Super_R":          gi.KeyRightSuper,
		"Tab":              gi.KeyTab,
		"ISO_Left_Tab":     gi.KeyTab,
		"Return":           gi.KeyReturn,
		"Up":               gi.KeyUpArrow,
		"Down":             gi.KeyDownArrow,
		"Left":             gi.KeyLeftArrow,
		"Right":            gi.KeyRightArrow,
		" ":                gi.KeySpace,
		"Escape":           gi.KeyEscape,
		"!":                gi.Key1,
		"@":                gi.Key2,
		"#":                gi.Key3,
		"$":                gi.Key4,
		"%":                gi.Key5,
		"^":                gi.Key6,
		"&":                gi.Key7,
		"*":                gi.Key8,
		"(":                gi.Key9,
		")":                gi.Key0,
		"_":                gi.KeyMinus,
		"+":                gi.KeyEqual,
		"|":                gi.KeyBackslash,
		"BackSpace":        gi.KeyBackspace,
		"Delete":           gi.KeyDelete,
		"quoteleft":        gi.KeyBackTick,
		"`":                gi.KeyBackTick,
		"~":                gi.KeyBackTick,
		"quoteright":       gi.KeyQuote,
		"\"":               gi.KeyQuote,
		"{":                gi.KeyLeftBracket,
		"}":                gi.KeyRightBracket,
		":":                gi.KeySemicolon,
		"<":                gi.KeyComma,
		">":                gi.KeyPeriod,
		"?":                gi.KeySlash,
		"F1":               gi.KeyF1,
		"F2":               gi.KeyF2,
		"F3":               gi.KeyF3,
		"F4":               gi.KeyF4,
		"F5":               gi.KeyF5,
		"F6":               gi.KeyF6,
		"F7":               gi.KeyF7,
		"F8":               gi.KeyF8,
		"F9":               gi.KeyF9,
		"F10":              gi.KeyF10,
		"F11":              gi.KeyF11,
		"F12":              gi.KeyF12,
		"F13":              gi.KeyF13,
		"F14":              gi.KeyF14,
		"F15":              gi.KeyF15,
		"F16":              gi.KeyF16,
		"L1":               gi.KeyF11,
		"L2":               gi.KeyF12,
		"XF86Tools":        gi.KeyF13,
		"XF86Launch5":      gi.KeyF14,
		"XF86Launch6":      gi.KeyF15,
		"XF86Launch7":      gi.KeyF16,
		"Num_Lock":         gi.KeyNumlock,
		"KP_Equal":         gi.KeyPadEqual,
		"Insert":           gi.KeyInsert,
		"Home":             gi.KeyHome,
		"Prior":            gi.KeyPrior,
		"Next":             gi.KeyNext,
		"Page_Up":          gi.KeyPrior,
		"Page_Down":        gi.KeyNext,
		"End":              gi.KeyEnd,
		"KP_Insert":        gi.KeyPadInsert,
		"KP_Delete":        gi.KeyPadDot,
		"KP_Enter":         gi.KeyPadEnter,
		"KP_End":           gi.KeyPadEnd,
		"KP_Down":          gi.KeyPadDown,
		"KP_Page_Down":     gi.KeyPadNext,
		"KP_Next":          gi.KeyPadNext,
		"KP_Left":          gi.KeyPadLeft,
		"KP_Begin":         gi.KeyPadBegin,
		"KP_Right":         gi.KeyPadRight,
		"KP_Home":          gi.KeyPadHome,
		"KP_Up":            gi.KeyPadUp,
		"KP_Prior":         gi.KeyPadPrior,
		"Caps_Lock":        gi.KeyCapsLock,
		"Terminate_Server": gi.KeyBackspace,
	}

	for l := byte('a'); l <= byte('z'); l++ {
		codeKeys[string(l)] = string(l)
		k := l + 'A' - 'a'
		codeKeys[string(k)] = string(l)
	}

}
