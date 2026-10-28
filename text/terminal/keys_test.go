// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package terminal

import (
	"testing"

	"cogentcore.org/core/events/key"
	"cogentcore.org/core/text/terminal/vt"
	"github.com/stretchr/testify/assert"
)

func mods(ms ...key.Modifiers) key.Modifiers {
	var m key.Modifiers
	for _, f := range ms {
		m.SetFlag(true, f)
	}
	return m
}

func TestKeyBytes(t *testing.T) {
	md := &vt.Modes{}
	app := &vt.Modes{AppCursorKeys: true, AppKeypad: true}
	tests := []struct {
		rn     rune
		code   key.Codes
		mods   key.Modifiers
		md     *vt.Modes
		altMet bool
		want   string
	}{
		{'a', key.CodeUnknown, 0, md, true, "a"},
		{'é', key.CodeUnknown, 0, md, true, "é"},
		{'A', key.CodeA, mods(key.Control), md, true, "\x01"},
		{'C', key.CodeC, mods(key.Control), md, true, "\x03"},
		{'[', key.CodeLeftSquareBracket, mods(key.Control), md, true, "\x1b"},
		{' ', key.CodeSpacebar, mods(key.Control), md, true, "\x00"},
		{'∫', key.CodeB, mods(key.Alt), md, true, "\x1bb"},
		{'∫', key.CodeB, mods(key.Alt), md, false, "∫"},
		{'B', key.CodeB, mods(key.Alt, key.Shift), md, true, "\x1bB"},
		{0, key.CodeReturnEnter, 0, md, true, "\r"},
		{0, key.CodeBackspace, 0, md, true, "\x7f"},
		{0, key.CodeBackspace, mods(key.Alt), md, true, "\x1b\x7f"},
		{0, key.CodeEscape, 0, md, true, "\x1b"},
		{0, key.CodeTab, 0, md, true, "\t"},
		{0, key.CodeTab, mods(key.Shift), md, true, "\x1b[Z"},
		{0, key.CodeUpArrow, 0, md, true, "\x1b[A"},
		{0, key.CodeUpArrow, 0, app, true, "\x1bOA"},
		{0, key.CodeLeftArrow, mods(key.Control), app, true, "\x1b[1;5D"},
		{0, key.CodeRightArrow, mods(key.Alt), md, true, "\x1b[1;3C"},
		{0, key.CodeHome, 0, md, true, "\x1b[H"},
		{0, key.CodeEnd, mods(key.Shift), md, true, "\x1b[1;2F"},
		{0, key.CodeDelete, 0, md, true, "\x1b[3~"},
		{0, key.CodePageDown, mods(key.Control), md, true, "\x1b[6;5~"},
		{0, key.CodeF1, 0, md, true, "\x1bOP"},
		{0, key.CodeF5, 0, md, true, "\x1b[15~"},
		{0, key.CodeF12, mods(key.Shift), md, true, "\x1b[24;2~"},
		{0, key.CodeKeypadEnter, 0, app, true, "\x1bOM"},
		{0, key.CodeCapsLock, 0, md, true, ""},
	}
	for _, tt := range tests {
		got := keyBytes(tt.rn, tt.code, tt.mods, tt.md, tt.altMet)
		assert.Equal(t, tt.want, string(got), "%q %v %v", tt.rn, tt.code, tt.mods)
	}
}

func TestMouseBytes(t *testing.T) {
	md := &vt.Modes{Mouse: vt.MouseNormal, MouseSGR: true}
	assert.Equal(t, "\x1b[<0;5;3M", string(mouseBytes(md, mousePress, mouseLeft, 2, 4, 0)))
	assert.Equal(t, "\x1b[<0;5;3m", string(mouseBytes(md, mouseRelease, mouseLeft, 2, 4, 0)))
	assert.Equal(t, "\x1b[<18;1;1M", string(mouseBytes(md, mousePress, mouseRight, 0, 0, mods(key.Control))))
	assert.Equal(t, "\x1b[<64;1;1M", string(mouseBytes(md, mousePress, mouseWheelUp, 0, 0, 0)))
	assert.Nil(t, mouseBytes(md, mouseMotion, mouseLeft, 0, 0, 0), "no motion in normal mode")
	md.Mouse = vt.MouseButton
	assert.Equal(t, "\x1b[<32;2;2M", string(mouseBytes(md, mouseMotion, mouseLeft, 1, 1, 0)))
	assert.Nil(t, mouseBytes(md, mouseMotion, mouseNoButton, 0, 0, 0))
	md.MouseSGR = false
	assert.Equal(t, "\x1b[M !!", string(mouseBytes(md, mousePress, mouseLeft, 0, 0, 0)))
	assert.Equal(t, "\x1b[M#!!", string(mouseBytes(md, mouseRelease, mouseLeft, 0, 0, 0)))
	md.Mouse = vt.MouseNone
	assert.Nil(t, mouseBytes(md, mousePress, mouseLeft, 0, 0, 0))
}

func TestCdCommand(t *testing.T) {
	assert.Equal(t, `cd '/a/it'\''s'`, cdCommand("bash", "/a/it's"))
	assert.Equal(t, `cd 'C:\it''s'`, cdCommand("powershell", `C:\it's`))
	assert.Equal(t, `cd /d "C:\dir"`, cdCommand("cmd", `C:\dir`))
}
