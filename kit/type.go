// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kit

// github.com/rcoreilly/goki/ki/kit

import (
	"bytes"
	"encoding/xml"
	"fmt"
	// "log"
	"reflect"
)

// Type provides JSON, XML marshal / unmarshal with encoding of underlying
// type name using kit.Types type name registry
type Type struct {
	T reflect.Type
}

// the full name of the type: package name + "." + type name
func (k Type) FullTypeName() string {
	return FullTypeName(k.T)
}

// stringer interface
func String(k Type) string {
	if k.T == nil {
		return "nil"
	}
	return FullTypeName(k.T)
}

// MarshalJSON saves only the type name
func (k Type) MarshalJSON() ([]byte, error) {
	if k.T == nil {
		b := []byte("null")
		return b, nil
	}
	nm := "\"" + k.FullTypeName() + "\""
	b := []byte(nm)
	return b, nil
}

// UnmarshalJSON loads the type name and looks it up in the Types registry of type names
func (k *Type) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		k.T = nil
		return nil
	}
	tn := string(bytes.Trim(bytes.TrimSpace(b), "\""))
	// fmt.Printf("loading type: %v", tn)
	typ := Types.Type(tn)
	if typ == nil {
		return fmt.Errorf("Type UnmarshalJSON: Types type name not found: %v", tn)
	}
	k.T = typ
	return nil
}

// todo: try to save info as an attribute within a single element instead of
// full start/end

// MarshalXML saves only the type name
func (k Type) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	tokens := []xml.Token{start}
	if k.T == nil {
		tokens = append(tokens, xml.CharData("null"))
	} else {
		tokens = append(tokens, xml.CharData(k.FullTypeName()))
	}
	tokens = append(tokens, xml.EndElement{start.Name})
	for _, t := range tokens {
		err := e.EncodeToken(t)
		if err != nil {
			return err
		}
	}
	err := e.Flush()
	if err != nil {
		return err
	}
	return nil
}

// UnmarshalXML loads the type name and looks it up in the Types registry of type names
func (k *Type) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	t, err := d.Token()
	if err != nil {
		return err
	}
	ct, ok := t.(xml.CharData)
	if ok {
		tn := string(bytes.TrimSpace([]byte(ct)))
		if tn == "null" {
			k.T = nil
		} else {
			// fmt.Printf("loading type: %v\n", tn)
			typ := Types.Type(tn)
			if typ == nil {
				return fmt.Errorf("Type UnmarshalXML: Types type name not found: %v", tn)
			}
			k.T = typ
		}
		t, err := d.Token()
		if err != nil {
			return err
		}
		et, ok := t.(xml.EndElement)
		if ok {
			if et.Name != start.Name {
				return fmt.Errorf("Type UnmarshalXML: EndElement: %v does not match StartElement: %v", et.Name, start.Name)
			}
			return nil
		}
		return fmt.Errorf("Type UnmarshalXML: Token: %+v is not expected EndElement", et)
	}
	return fmt.Errorf("Type UnmarshalXML: Token: %+v is not expected EndElement", ct)
}
