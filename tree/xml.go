// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree

import (
	"encoding/xml"
	"io"
	"log"

	"cogentcore.org/core/base/errors"
)

// WriteXML writes the tree to an XML-encoded byte string over io.Writer
// using MarshalXML.
func (n *NodeBase) WriteXML(writer io.Writer, indent bool) error {
	err := checkThis(n)
	if err != nil {
		return err
	}
	var b []byte
	if indent {
		b, err = xml.MarshalIndent(n.This(), "", "  ")
	} else {
		b, err = xml.Marshal(n.This())
	}
	if err != nil {
		return errors.Log(err)
	}
	_, err = writer.Write(b)
	if err != nil {
		return errors.Log(err)
	}
	return nil
}

// ReadXML reads the tree from an XML-encoded byte string over io.Reader, calls
// UnmarshalPost to recover pointers from paths.
func (n *NodeBase) ReadXML(reader io.Reader) error {
	var err error
	if err = checkThis(n); err != nil {
		log.Println(err)
		return err
	}
	b, err := io.ReadAll(reader)
	if err != nil {
		log.Println(err)
		return err
	}
	err = xml.Unmarshal(b, n.This()) // key use of this!
	UnmarshalPost(n.This())
	return err
}

// todo: save N as an attr instead of a full element

/*
// MarshalXML saves the length and type information for each object in a
// slice, as a separate struct-like record at the start, followed by the
// structs for each element in the slice -- this allows the Unmarshal to first
// create all the elements and then load them
func (sl Slice) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	tokens := []xml.Token{start}
	nk := len(sl)
	nt := xml.StartElement{Name: xml.Name{Space: "", Local: "N"}}
	tokens = append(tokens, nt, xml.CharData(fmt.Sprintf("%d", nk)), xml.EndElement{Name: nt.Name})
	for _, kid := range sl {
		knm := kid.NodeType().Name
		t := xml.StartElement{Name: xml.Name{Space: "", Local: "Type"}}
		tokens = append(tokens, t, xml.CharData(knm), xml.EndElement{Name: t.Name})
	}
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
	for _, kid := range sl {
		knm := reflect.TypeOf(kid).Elem().Name()
		ct := xml.StartElement{Name: xml.Name{Space: "", Local: knm}}
		err := e.EncodeElement(kid, ct)
		if err != nil {
			return err
		}
	}
	err = e.EncodeToken(xml.EndElement{Name: start.Name})
	if err != nil {
		return err
	}
	err = e.Flush()
	if err != nil {
		return err
	}
	return nil
}

// DecodeXMLStartEl reads a start element token
func DecodeXMLStartEl(d *xml.Decoder) (start xml.StartElement, err error) {
	for {
		var t xml.Token
		t, err = d.Token()
		if err != nil {
			log.Printf("tree.DecodeXMLStartEl err %v\n", err)
			return
		}
		switch tv := t.(type) {
		case xml.StartElement:
			start = tv
			return
		case xml.CharData: // actually passes the spaces and everything through here
			continue
		case xml.EndElement:
			err = fmt.Errorf("tree.DecodeXMLStartEl: got unexpected EndElement")
			errors.Log(err)
			return
		default:
			continue
		}
	}
}

// DecodeXMLEndEl reads an end element
func DecodeXMLEndEl(d *xml.Decoder, start xml.StartElement) error {
	for {
		t, err := d.Token()
		if err != nil {
			log.Printf("tree.DecodeXMLEndEl err %v\n", err)
			return err
		}
		switch tv := t.(type) {
		case xml.EndElement:
			if tv.Name != start.Name {
				err = fmt.Errorf("tree.DecodeXMLEndEl: EndElement: %v does not match StartElement: %v", tv.Name, start.Name)
				errors.Log(err)
				return err
			}
			return nil
		case xml.CharData: // actually passes the spaces and everything through here
			continue
		case xml.StartElement:
			err = fmt.Errorf("tree.DecodeXMLEndEl: got unexpected StartElement: %v", tv.Name)
			errors.Log(err)
			return err
		default:
			continue
		}
	}
}

// DecodeXMLCharData reads char data..
func DecodeXMLCharData(d *xml.Decoder) (val string, err error) {
	for {
		var t xml.Token
		t, err = d.Token()
		if err != nil {
			log.Printf("tree.DecodeXMLCharData err %v\n", err)
			return
		}
		switch tv := t.(type) {
		case xml.CharData:
			val = string([]byte(tv))
			return
		case xml.StartElement:
			err = fmt.Errorf("tree.DecodeXMLCharData: got unexpected StartElement: %v", tv.Name)
			errors.Log(err)
			return
		case xml.EndElement:
			err = fmt.Errorf("tree.DecodeXMLCharData: got unexpected EndElement: %v", tv.Name)
			errors.Log(err)
			return
		}
	}
}

// DecodeXMLCharEl reads a start / chardata / end sequence of 3 elements, returning name, val
func DecodeXMLCharEl(d *xml.Decoder) (name, val string, err error) {
	var st xml.StartElement
	st, err = DecodeXMLStartEl(d)
	if err != nil {
		return
	}
	name = st.Name.Local
	val, err = DecodeXMLCharData(d)
	if err != nil {
		return
	}
	err = DecodeXMLEndEl(d, st)
	if err != nil {
		return
	}
	return
}

// UnmarshalXML parses the length and type information for each object in the
// slice, creates the new slice with those elements, and then loads based on
// the remaining bytes which represent each element
func (sl *Slice) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	// for _, attr := range start.Attr {
	// 	// todo: need to set the properties from name / value -- don't have parent though!
	// }
	name, val, err := DecodeXMLCharEl(d)
	if err != nil {
		return err
	}
	if name == "N" {
		n64, err := strconv.ParseInt(string(val), 10, 64)
		if err != nil {
			return err
		}
		n := int(n64)
		if n == 0 {
			return DecodeXMLEndEl(d, start)
		}
		// fmt.Printf("n parsed: %d from %v\n", n, string(val))
		nwk := make([]Node, 0, n) // allocate new slice

		for i := 0; i < n; i++ {
			name, val, err = DecodeXMLCharEl(d)
			if err != nil {
				return errors.Log(err)
			}
			if name == "Type" {
				tn := strings.TrimSpace(val)
				typ, err := types.TypeByNameTry(tn)
				if typ == nil {
					return fmt.Errorf("tree.Slice UnmarshalXML: %w", err)
				}
				kid := NewOfType(typ)
				initNode(kid)
				nwk = append(nwk, kid)
			}
		}

		for i := 0; i < n; i++ {
			st, err := DecodeXMLStartEl(d)
			if err != nil {
				return err
			}
			// todo: could double-check st
			err = d.DecodeElement(nwk[i], &st)
			if err != nil {
				log.Printf("%v", err)
				return err
			}
		}
		*sl = append(*sl, nwk...)
		// } else {
	}
	// todo: in theory we could just parse a list of type names as tags, but for the "dump" format
	// this is more robust.
	return DecodeXMLEndEl(d, start) // final end
}
*/
