// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux && !android

package desktop

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"

	"cogentcore.org/core/events/key"
	"cogentcore.org/core/system"
	"github.com/go-gl/glfw/v3.4/glfw"
	"github.com/godbus/dbus/v5"
)

// Input method (IME) support on Linux, with IBus or Fcitx5 over D-Bus,
// through the IBus portal on the session bus, which both provide (as used
// by sandboxed apps), or else the IBus bus. While a text widget is focused,
// keys are sent to the input method first, and those that it handles are
// not sent as key events; its preedit (composition) and committed text are
// sent as events. GLFW's own XIM input method support is turned off when
// this is used, as it would also handle the keys.

const ibusInputContext = "org.freedesktop.IBus.InputContext"

// theIME is the input method connection, or nil if there is none.
var theIME *ibusIME

// ibusIME is a connection to an input method with the IBus D-Bus protocol.
type ibusIME struct {
	conn *dbus.Conn
	ic   dbus.BusObject // the input context

	// mu protects the fields below, which are used by
	// the signal handler goroutine.
	mu sync.Mutex

	// win is the window with the text input, or nil if there is none.
	win *Window

	// composing is whether a composition has been sent.
	composing bool
}

// suppressChar is whether the next char callback is for a key that the
// input method handled, so it is not sent. It is only used on the main thread.
var suppressChar bool

// imeInitApp connects to the input method if there is one, before GLFW is
// initialized, so that GLFW does not also use it through XIM, which Xlib
// sets up from XMODIFIERS when GLFW is initialized. The returned function
// restores XMODIFIERS after that, for any processes that the app starts.
func imeInitApp() (restore func()) {
	restore = func() {}
	if os.Getenv("COGENT_IME") == "none" {
		return
	}
	ime, err := connectIME()
	if err != nil { // no input method: GLFW still gets committed text through XIM
		return
	}
	theIME = ime
	old, had := os.LookupEnv("XMODIFIERS")
	os.Setenv("XMODIFIERS", "@im=none")
	return func() {
		if had {
			os.Setenv("XMODIFIERS", old)
		} else {
			os.Unsetenv("XMODIFIERS")
		}
	}
}

// connectIME connects to the input method through the IBus portal on the
// session bus, or else the IBus bus, and creates an input context.
func connectIME() (*ibusIME, error) {
	ime := &ibusIME{}
	conn, err := dbus.ConnectSessionBus(dbus.WithSignalHandler(ime))
	if err == nil {
		var path dbus.ObjectPath
		err = conn.Object("org.freedesktop.portal.IBus", "/org/freedesktop/IBus").
			Call("org.freedesktop.IBus.Portal.CreateInputContext", 0, "cogentcore").Store(&path)
		if err == nil {
			return ime.init(conn, "org.freedesktop.portal.IBus", path)
		}
		conn.Close()
	}
	addr, err := ibusAddress()
	if err != nil {
		return nil, err
	}
	conn, err = dbus.Connect(addr, dbus.WithSignalHandler(ime))
	if err != nil {
		return nil, err
	}
	var path dbus.ObjectPath
	err = conn.Object("org.freedesktop.IBus", "/org/freedesktop/IBus").
		Call("org.freedesktop.IBus.CreateInputContext", 0, "cogentcore").Store(&path)
	if err != nil {
		conn.Close()
		return nil, err
	}
	return ime.init(conn, "org.freedesktop.IBus", path)
}

// init sets up the input context with the given bus name and path.
func (ime *ibusIME) init(conn *dbus.Conn, name string, path dbus.ObjectPath) (*ibusIME, error) {
	const (
		capPreeditText = 1 << 0
		capFocus       = 1 << 3
	)
	ime.conn = conn
	ime.ic = conn.Object(name, path)
	if err := ime.ic.Call(ibusInputContext+".SetCapabilities", 0, uint32(capPreeditText|capFocus)).Err; err != nil {
		conn.Close()
		return nil, err
	}
	if err := conn.AddMatchSignal(dbus.WithMatchObjectPath(path), dbus.WithMatchInterface(ibusInputContext)); err != nil {
		conn.Close()
		return nil, err
	}
	return ime, nil
}

// ibusAddress returns the address of the IBus bus, from IBUS_ADDRESS
// or the file that IBus writes for the display.
func ibusAddress() (string, error) {
	if addr := os.Getenv("IBUS_ADDRESS"); addr != "" {
		return addr, nil
	}
	mid, err := os.ReadFile("/etc/machine-id")
	if err != nil {
		return "", err
	}
	host, disp, ok := strings.Cut(os.Getenv("DISPLAY"), ":")
	if !ok || host != "" {
		return "", fmt.Errorf("ibus: no local X display")
	}
	disp, _, _ = strings.Cut(disp, ".")
	conf := os.Getenv("XDG_CONFIG_HOME")
	if conf == "" {
		conf = filepath.Join(os.Getenv("HOME"), ".config")
	}
	b, err := os.ReadFile(filepath.Join(conf, "ibus", "bus", fmt.Sprintf("%s-unix-%s", strings.TrimSpace(string(mid)), disp)))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(b), "\n") {
		if addr, ok := strings.CutPrefix(line, "IBUS_ADDRESS="); ok {
			return addr, nil
		}
	}
	return "", fmt.Errorf("ibus: no address")
}

// initIME sets up input methods (IME) for the window, which uses
// the connection of the app.
func (w *Window) initIME() {}

func (w *Window) SetTextInput(ti *system.TextInput) {
	ime := theIME
	if ime == nil {
		return
	}
	ime.mu.Lock()
	if ti == nil {
		if ime.win == w {
			ime.win = nil
			ime.endComposition()
			ime.mu.Unlock()
			ime.ic.Go(ibusInputContext+".Reset", 0, nil)
			ime.ic.Go(ibusInputContext+".FocusOut", 0, nil)
			return
		}
		ime.mu.Unlock()
		return
	}
	focusIn := ime.win != w
	ime.win = w
	ime.mu.Unlock()
	if focusIn {
		ime.ic.Go(ibusInputContext+".FocusIn", 0, nil)
	}
	// The cursor location is in screen pixels, from the window position,
	// which is not known on Wayland, where it is relative to the window.
	r := ti.Caret
	x, y := r.Min.X, r.Min.Y
	if glfw.GetPlatform() == glfw.PlatformX11 {
		x, y = x+w.Pos.X, y+w.Pos.Y
	}
	ime.ic.Go(ibusInputContext+".SetCursorLocation", 0, nil, int32(x), int32(y), int32(r.Dx()), int32(r.Dy()))
}

// imeFocus is called when the window gets or loses the focus.
func (w *Window) imeFocus(focused bool) {
	ime := theIME
	if ime == nil {
		return
	}
	ime.mu.Lock()
	has := ime.win == w
	ime.mu.Unlock()
	if !has {
		return
	}
	if focused {
		ime.ic.Go(ibusInputContext+".FocusIn", 0, nil)
	} else {
		ime.ic.Go(ibusInputContext+".FocusOut", 0, nil)
	}
}

// imeKey sends the key event to the input method while text input is
// active, returning true if it handled the key, which is then not sent as
// a key event, nor its char. It is called on the main thread.
func (w *Window) imeKey(ky glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) bool {
	suppressChar = false
	ime := theIME
	if ime == nil {
		return false
	}
	ime.mu.Lock()
	active := ime.win == w
	ime.mu.Unlock()
	if !active {
		return false
	}
	sym := keysym(ky, scancode, mods)
	if sym == 0 {
		return false
	}
	state := ibusState(mods)
	if action == glfw.Release {
		state |= 1 << 30 // IBUS_RELEASE_MASK
	}
	var handled bool
	if err := ime.ic.Call(ibusInputContext+".ProcessKeyEvent", 0, sym, uint32(max(scancode-8, 0)), state).Store(&handled); err != nil {
		return false
	}
	if handled && action != glfw.Release {
		suppressChar = true
	}
	return handled
}

// imeChar returns whether the char callback is for a key that
// the input method handled, so it is not sent.
func (w *Window) imeChar() bool {
	s := suppressChar
	suppressChar = false
	return s
}

// ibusState returns the X11 modifier state for the given modifiers.
func ibusState(mods glfw.ModifierKey) uint32 {
	var st uint32
	if mods&glfw.ModShift != 0 {
		st |= 1 // ShiftMask
	}
	if mods&glfw.ModCapsLock != 0 {
		st |= 2 // LockMask
	}
	if mods&glfw.ModControl != 0 {
		st |= 4 // ControlMask
	}
	if mods&glfw.ModAlt != 0 {
		st |= 8 // Mod1Mask
	}
	if mods&glfw.ModNumLock != 0 {
		st |= 16 // Mod2Mask
	}
	if mods&glfw.ModSuper != 0 {
		st |= 64 // Mod4Mask
	}
	return st
}

// keysym returns the X11 keysym of the given key, which is the char of a
// printable key in the keyboard layout, or 0 if it is not known.
func keysym(ky glfw.Key, scancode int, mods glfw.ModifierKey) uint32 {
	if sym, ok := specialKeysyms[ky]; ok {
		return sym
	}
	name := glfw.GetKeyName(ky, scancode)
	rs := []rune(name)
	if len(rs) != 1 {
		return 0
	}
	r := rs[0]
	if mods&glfw.ModShift != 0 != (mods&glfw.ModCapsLock != 0) {
		r = unicode.ToUpper(r)
	}
	if r < 0x100 { // Latin-1 keysyms are the same as the code points
		return uint32(r)
	}
	return 0x01000000 | uint32(r)
}

// specialKeysyms are the X11 keysyms of the keys that are not printable.
var specialKeysyms = map[glfw.Key]uint32{
	glfw.KeySpace:        0x0020,
	glfw.KeyBackspace:    0xff08,
	glfw.KeyTab:          0xff09,
	glfw.KeyEnter:        0xff0d,
	glfw.KeyPause:        0xff13,
	glfw.KeyScrollLock:   0xff14,
	glfw.KeyEscape:       0xff1b,
	glfw.KeyHome:         0xff50,
	glfw.KeyLeft:         0xff51,
	glfw.KeyUp:           0xff52,
	glfw.KeyRight:        0xff53,
	glfw.KeyDown:         0xff54,
	glfw.KeyPageUp:       0xff55,
	glfw.KeyPageDown:     0xff56,
	glfw.KeyEnd:          0xff57,
	glfw.KeyInsert:       0xff63,
	glfw.KeyMenu:         0xff67,
	glfw.KeyNumLock:      0xff7f,
	glfw.KeyKPEnter:      0xff8d,
	glfw.KeyDelete:       0xffff,
	glfw.KeyLeftShift:    0xffe1,
	glfw.KeyRightShift:   0xffe2,
	glfw.KeyLeftControl:  0xffe3,
	glfw.KeyRightControl: 0xffe4,
	glfw.KeyCapsLock:     0xffe5,
	glfw.KeyLeftAlt:      0xffe9,
	glfw.KeyRightAlt:     0xffea,
	glfw.KeyLeftSuper:    0xffeb,
	glfw.KeyRightSuper:   0xffec,
	glfw.KeyF1:           0xffbe,
	glfw.KeyF2:           0xffbf,
	glfw.KeyF3:           0xffc0,
	glfw.KeyF4:           0xffc1,
	glfw.KeyF5:           0xffc2,
	glfw.KeyF6:           0xffc3,
	glfw.KeyF7:           0xffc4,
	glfw.KeyF8:           0xffc5,
	glfw.KeyF9:           0xffc6,
	glfw.KeyF10:          0xffc7,
	glfw.KeyF11:          0xffc8,
	glfw.KeyF12:          0xffc9,
}

// endComposition sends the end of the composition, if one was sent.
// It must be called with mu locked.
func (ime *ibusIME) endComposition() {
	if ime.composing && ime.win != nil {
		ime.composing = false
		ime.win.Event.Composition("", 0, 0)
		glfw.PostEmptyEvent()
	}
	ime.composing = false
}

// DeliverSignal handles the signals of the input context, on the
// goroutine of the D-Bus connection.
func (ime *ibusIME) DeliverSignal(iface, name string, signal *dbus.Signal) {
	if iface != ibusInputContext {
		return
	}
	ime.mu.Lock()
	defer ime.mu.Unlock()
	w := ime.win
	if w == nil {
		return
	}
	switch name {
	case "CommitText":
		txt, _, _ := ibusText(signal.Body, 0)
		ime.endComposition()
		for _, r := range txt {
			w.Event.KeyChord(r, key.CodeUnknown, 0)
		}
		glfw.PostEmptyEvent()
	case "UpdatePreeditText", "UpdatePreeditTextWithMode":
		txt, start, end := ibusText(signal.Body, 0)
		visible := len(signal.Body) > 2 && signal.Body[2] == true
		if !visible || txt == "" {
			ime.endComposition()
			return
		}
		if start < 0 { // the cursor
			cur, _ := signal.Body[1].(uint32)
			start, end = int(cur), int(cur)
		}
		ime.composing = true
		w.Event.Composition(txt, start, end)
		glfw.PostEmptyEvent()
	case "HidePreeditText":
		ime.endComposition()
	}
}

// ibusText returns the text of the IBusText at the given index of the
// signal body, and the range of its double underlined attribute, which is
// the clause being converted, or -1 if there is none.
func ibusText(body []any, i int) (txt string, start, end int) {
	start, end = -1, -1
	if len(body) <= i {
		return
	}
	v, ok := body[i].(dbus.Variant)
	if !ok {
		return
	}
	// IBusText: (s a{sv} s v), with the attribute list v: (s a{sv} av)
	fields, ok := v.Value().([]any)
	if !ok || len(fields) < 3 {
		return
	}
	txt, _ = fields[2].(string)
	if len(fields) < 4 {
		return
	}
	al, ok := fields[3].(dbus.Variant)
	if !ok {
		return
	}
	alf, ok := al.Value().([]any)
	if !ok || len(alf) < 3 {
		return
	}
	attrs, ok := alf[2].([]dbus.Variant)
	if !ok {
		return
	}
	for _, a := range attrs {
		// IBusAttribute: (s a{sv} u u u u): type, value, start, end
		af, ok := a.Value().([]any)
		if !ok || len(af) < 6 {
			continue
		}
		typ, _ := af[2].(uint32)
		val, _ := af[3].(uint32)
		st, _ := af[4].(uint32)
		ed, _ := af[5].(uint32)
		const (
			attrUnderline   = 1
			underlineDouble = 2
		)
		if typ == attrUnderline && val == underlineDouble {
			return txt, int(st), int(ed)
		}
	}
	return
}
