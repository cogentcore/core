// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux && !android

package desktop

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/godbus/dbus/v5"
)

// TestIBusText checks parsing of an IBusText as decoded from the wire.
func TestIBusText(t *testing.T) {
	type attr struct {
		Name                   string
		Attach                 map[string]dbus.Variant
		Type, Value, Start, End uint32
	}
	type attrList struct {
		Name   string
		Attach map[string]dbus.Variant
		Attrs  []dbus.Variant
	}
	type text struct {
		Name   string
		Attach map[string]dbus.Variant
		Text   string
		Attrs  dbus.Variant
	}
	al := attrList{"IBusAttrList", map[string]dbus.Variant{}, []dbus.Variant{
		dbus.MakeVariant(attr{"IBusAttribute", map[string]dbus.Variant{}, 1, 1, 0, 3}), // single underline
		dbus.MakeVariant(attr{"IBusAttribute", map[string]dbus.Variant{}, 1, 2, 1, 3}), // double: the clause
	}}
	tx := text{"IBusText", map[string]dbus.Variant{}, "にほん", dbus.MakeVariant(al)}
	msg := &dbus.Message{Type: dbus.TypeSignal, Headers: map[dbus.HeaderField]dbus.Variant{
		dbus.FieldPath:      dbus.MakeVariant(dbus.ObjectPath("/x")),
		dbus.FieldInterface: dbus.MakeVariant(ibusInputContext),
		dbus.FieldMember:    dbus.MakeVariant("UpdatePreeditText"),
	}, Body: []any{dbus.MakeVariant(tx), uint32(2), true}}
	msg.Headers[dbus.FieldSignature] = dbus.MakeVariant(dbus.SignatureOf(msg.Body...))
	var buf bytes.Buffer
	if err := msg.EncodeTo(&buf, binary.LittleEndian); err != nil {
		t.Fatal(err)
	}
	dm, err := dbus.DecodeMessage(&buf)
	if err != nil {
		t.Fatal(err)
	}
	txt, st, ed := ibusText(dm.Body, 0)
	if txt != "にほん" || st != 1 || ed != 3 {
		t.Errorf("got %q %d %d", txt, st, ed)
	}
	if dm.Body[1].(uint32) != 2 || dm.Body[2] != true {
		t.Errorf("cursor/visible: %v", dm.Body[1:])
	}
}
