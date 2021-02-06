// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/goki/ki/kit"
)

// see https://github.com/goki/ki/wiki/Naming for IO naming conventions

// TODO: switch to using Decode / Encode instead of
// [Un]MarshalJSON which uses byte[] instead of io.Reader / Writer..

// JSONTypePrefix is the first thing output in a ki tree JSON output file,
// specifying the type of the root node of the ki tree -- this info appears
// all on one { } bracketed line at the start of the file, and can also be
// used to identify the file as a ki tree JSON file
var JSONTypePrefix = []byte("{\"ki.RootType\": ")

// JSONTypeSuffix is just the } and \n at the end of the prefix line
var JSONTypeSuffix = []byte("}\n")

// WriteJSON writes the tree to an io.Writer, using MarshalJSON -- also
// saves a critical starting record that allows file to be loaded de-novo
// and recreate the proper root type for the tree.
// This calls UniquifyNamesAll because it is essential that names be unique
// at this point.
func (n *Node) WriteJSON(writer io.Writer, indent bool) error {
	err := ThisCheck(n)
	if err != nil {
		return err
	}
	UniquifyNamesAll(n.This())
	var b []byte
	if indent {
		b, err = json.MarshalIndent(n.This(), "", "  ")
	} else {
		b, err = json.Marshal(n.This())
	}
	if err != nil {
		log.Println(err)
		return err
	}
	knm := kit.Types.TypeName(Type(n.This()))
	tstr := string(JSONTypePrefix) + fmt.Sprintf("\"%v\"}\n", knm)
	nwb := make([]byte, len(b)+len(tstr))
	copy(nwb, []byte(tstr))
	copy(nwb[len(tstr):], b) // is there a way to avoid this?
	_, err = writer.Write(nwb)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// SaveJSON saves the tree to a JSON-encoded file, using WriteJSON.
func (n *Node) SaveJSON(filename string) error {
	fp, err := os.Create(filename)
	defer fp.Close()
	if err != nil {
		log.Println(err)
		return err
	}
	err = n.WriteJSON(bufio.NewWriter(fp), Indent) // use indent by default
	if err != nil {
		log.Println(err)
	}
	return err
}

// ReadJSON reads and unmarshals tree starting at this node, from a
// JSON-encoded byte stream via io.Reader.  First element in the stream
// must be of same type as this node -- see ReadNewJSON function to
// construct a new tree.  Uses ConfigureChildren to minimize changes from
// current tree relative to loading one -- wraps UnmarshalJSON and calls
// UnmarshalPost to recover pointers from paths.
func (n *Node) ReadJSON(reader io.Reader) error {
	err := ThisCheck(n)
	if err != nil {
		log.Println(err)
		return err
	}
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Println(err)
		return err
	}
	updt := n.UpdateStart()
	stidx := 0
	if bytes.HasPrefix(b, JSONTypePrefix) { // skip type
		stidx = bytes.Index(b, JSONTypeSuffix) + len(JSONTypeSuffix)
	}
	// todo: use json.NewDecoder, Decode instead -- need to deal with TypePrefix etc above
	err = json.Unmarshal(b[stidx:], n.This()) // key use of this!
	if err == nil {
		n.UnmarshalPost()
	}
	n.SetFlag(int(ChildAdded)) // this might not be set..
	n.UpdateEnd(updt)
	return err
}

// OpenJSON opens file over this tree from a JSON-encoded file -- see
// ReadJSON for details, and OpenNewJSON for opening an entirely new tree.
func (n *Node) OpenJSON(filename string) error {
	fp, err := os.Open(filename)
	defer fp.Close()
	if err != nil {
		log.Println(err)
		return err
	}
	return n.ReadJSON(bufio.NewReader(fp))
}

// ReadNewJSON reads a new Ki tree from a JSON-encoded byte string, using type
// information at start of file to create an object of the proper type
func ReadNewJSON(reader io.Reader) (Ki, error) {
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if bytes.HasPrefix(b, JSONTypePrefix) {
		stidx := len(JSONTypePrefix) + 1
		eidx := bytes.Index(b, JSONTypeSuffix)
		bodyidx := eidx + len(JSONTypeSuffix)
		tn := string(bytes.Trim(bytes.TrimSpace(b[stidx:eidx]), "\""))
		typ := kit.Types.Type(tn)
		if typ == nil {
			return nil, fmt.Errorf("ki.OpenNewJSON: kit.Types type name not found: %v", tn)
		}
		root := NewOfType(typ)
		InitNode(root)

		updt := root.UpdateStart()
		err = json.Unmarshal(b[bodyidx:], root)
		if err == nil {
			root.UnmarshalPost()
		}
		root.SetFlag(int(ChildAdded)) // this might not be set..
		root.UpdateEnd(updt)
		return root, nil
	}
	return nil, fmt.Errorf("ki.OpenNewJSON -- type prefix not found at start of file -- must be there to identify type of root node of tree")
}

// OpenNewJSON opens a new Ki tree from a JSON-encoded file, using type
// information at start of file to create an object of the proper type
func OpenNewJSON(filename string) (Ki, error) {
	fp, err := os.Open(filename)
	defer fp.Close()
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return ReadNewJSON(bufio.NewReader(fp))
}

// WriteXML writes the tree to an XML-encoded byte string over io.Writer
// using MarshalXML.
func (n *Node) WriteXML(writer io.Writer, indent bool) error {
	err := ThisCheck(n)
	if err != nil {
		log.Println(err)
		return err
	}
	var b []byte
	if indent {
		b, err = xml.MarshalIndent(n.This(), "", "  ")
	} else {
		b, err = xml.Marshal(n.This())
	}
	if err != nil {
		log.Println(err)
		return err
	}
	_, err = writer.Write(b)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// ReadXML reads the tree from an XML-encoded byte string over io.Reader, calls
// UnmarshalPost to recover pointers from paths.
func (n *Node) ReadXML(reader io.Reader) error {
	var err error
	if err = ThisCheck(n); err != nil {
		log.Println(err)
		return err
	}
	b, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Println(err)
		return err
	}
	updt := n.UpdateStart()
	err = xml.Unmarshal(b, n.This()) // key use of this!
	if err == nil {
		n.UnmarshalPost()
	}
	n.SetFlag(int(ChildAdded)) // this might not be set..
	n.UpdateEnd(updt)
	return nil
}

// ParentAllChildren walks the tree down from current node and call
// SetParent on all children -- needed after an Unmarshal.
func (n *Node) ParentAllChildren() {
	for _, child := range *n.Children() {
		if child != nil {
			child.AsNode().Par = n.This()
			child.ParentAllChildren()
		}
	}
}

// UnmarshalPost must be called after an Unmarshal -- calls
// ParentAllChildren.
func (n *Node) UnmarshalPost() {
	n.ParentAllChildren()
}

//////////////////////////////////////////////////////////////////////////
// Slice

// MarshalJSON saves the length and type, name information for each object in a
// slice, as a separate struct-like record at the start, followed by the
// structs for each element in the slice -- this allows the Unmarshal to first
// create all the elements and then load them
func (sl Slice) MarshalJSON() ([]byte, error) {
	nk := len(sl)
	b := make([]byte, 0, nk*100+20)
	if nk == 0 {
		b = append(b, []byte("null")...)
		return b, nil
	}
	nstr := fmt.Sprintf("[{\"n\":%d,", nk)
	b = append(b, []byte(nstr)...)
	for i, kid := range sl {
		// fmt.Printf("json out of %v\n", kid.Path())
		knm := kit.Types.TypeName(reflect.TypeOf(kid).Elem())
		tstr := fmt.Sprintf("\"type\":\"%v\", \"name\": \"%v\"", knm, kid.Name()) // todo: escape names!
		b = append(b, []byte(tstr)...)
		if i < nk-1 {
			b = append(b, []byte(",")...)
		}
	}
	b = append(b, []byte("},")...)
	for i, kid := range sl {
		var err error
		var kb []byte
		kb, err = json.Marshal(kid)
		if err == nil {
			b = append(b, []byte("{")...)
			b = append(b, kb[1:len(kb)-1]...)
			b = append(b, []byte("}")...)
			if i < nk-1 {
				b = append(b, []byte(",")...)
			}
		} else {
			fmt.Printf("error doing json.Marshall from kid: %v\n", kid.Path())
			log.Println(err)
			fmt.Printf("output to point of error: %v\n", string(b))
		}
	}
	b = append(b, []byte("]")...)
	// fmt.Printf("json out: %v\n", string(b))
	return b, nil
}

///////////////////////////////////////////////////////////////////////////
// JSON

// UnmarshalJSON parses the length and type information for each object in the
// slice, creates the new slice with those elements, and then loads based on
// the remaining bytes which represent each element
func (sl *Slice) UnmarshalJSON(b []byte) error {
	// fmt.Printf("json in: %v\n", string(b))
	if bytes.Equal(b, []byte("null")) {
		*sl = nil
		return nil
	}
	lb := bytes.IndexRune(b, '{')
	rb := bytes.IndexRune(b, '}')
	if lb < 0 || rb < 0 { // probably null
		return nil
	}
	// todo: if name contains "," this won't work..
	flds := bytes.Split(b[lb+1:rb], []byte(","))
	if len(flds) == 0 {
		return errors.New("Slice UnmarshalJSON: no child data found")
	}
	// fmt.Printf("flds[0]:\n%v\n", string(flds[0]))
	ns := bytes.Index(flds[0], []byte("\"n\":"))
	bn := bytes.TrimSpace(flds[0][ns+4:])

	n64, err := strconv.ParseInt(string(bn), 10, 64)
	if err != nil {
		return err
	}
	n := int(n64)
	if n == 0 {
		return nil
	}
	// fmt.Printf("n parsed: %d from %v\n", n, string(bn))

	tnl := make(kit.TypeAndNameList, n)

	for i := 0; i < n; i++ {
		fld := flds[2*i+1]
		// fmt.Printf("fld:\n%v\n", string(fld))
		ti := bytes.Index(fld, []byte("\"type\":"))
		tn := string(bytes.Trim(bytes.TrimSpace(fld[ti+7:]), "\""))
		fld = flds[2*i+2]
		ni := bytes.Index(fld, []byte("\"name\":"))
		nm := string(bytes.Trim(bytes.TrimSpace(fld[ni+7:]), "\""))
		// fmt.Printf("making type: %v", tn)
		typ := kit.Types.Type(tn)
		if typ == nil {
			return fmt.Errorf("ki.Slice UnmarshalJSON: kit.Types type name not found: %v", tn)
		}
		tnl[i].Type = typ
		tnl[i].Name = nm
	}

	sl.Config(nil, tnl)

	nwk := make([]Ki, n) // allocate new slice containing *pointers* to kids

	for i, kid := range *sl {
		nwk[i] = kid
	}

	cb := make([]byte, 0, 1+len(b)-rb)
	cb = append(cb, []byte("[")...)
	cb = append(cb, b[rb+2:]...)

	// fmt.Printf("loading:\n%v", string(cb))

	err = json.Unmarshal(cb, &nwk)
	if err != nil {
		return err
	}
	return nil
}

// todo: save N as an attr instead of a full element

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
		knm := kit.Types.TypeName(reflect.TypeOf(kid).Elem())
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
			log.Printf("ki.DecodeXMLStartEl err %v\n", err)
			return
		}
		switch tv := t.(type) {
		case xml.StartElement:
			start = tv
			return
		case xml.CharData: // actually passes the spaces and everything through here
			continue
		case xml.EndElement:
			err = fmt.Errorf("ki.DecodeXMLStartEl: got unexpected EndElement")
			log.Println(err)
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
			log.Printf("ki.DecodeXMLEndEl err %v\n", err)
			return err
		}
		switch tv := t.(type) {
		case xml.EndElement:
			if tv.Name != start.Name {
				err = fmt.Errorf("ki.DecodeXMLEndEl: EndElement: %v does not match StartElement: %v", tv.Name, start.Name)
				log.Println(err)
				return err
			}
			return nil
		case xml.CharData: // actually passes the spaces and everything through here
			continue
		case xml.StartElement:
			err = fmt.Errorf("ki.DecodeXMLEndEl: got unexpected StartElement: %v", tv.Name)
			log.Println(err)
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
			log.Printf("ki.DecodeXMLCharData err %v\n", err)
			return
		}
		switch tv := t.(type) {
		case xml.CharData:
			val = string([]byte(tv))
			return
		case xml.StartElement:
			err = fmt.Errorf("ki.DecodeXMLCharData: got unexpected StartElement: %v", tv.Name)
			log.Println(err)
			return
		case xml.EndElement:
			err = fmt.Errorf("ki.DecodeXMLCharData: got unexpected EndElement: %v", tv.Name)
			log.Println(err)
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
	// 	// todo: need to set the props from name / value -- don't have parent though!
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
		nwk := make([]Ki, 0, n) // allocate new slice

		for i := 0; i < n; i++ {
			name, val, err = DecodeXMLCharEl(d)
			if name == "Type" {
				tn := strings.TrimSpace(val)
				// fmt.Printf("making type: %v\n", tn)
				typ := kit.Types.Type(tn)
				if typ == nil {
					return fmt.Errorf("ki.Slice UnmarshalXML: kit.Types type name not found: %v", tn)
				}
				nkid := reflect.New(typ).Interface()
				// fmt.Printf("nkid is new obj of type %T val: %+v\n", nkid, nkid)
				kid, ok := nkid.(Ki)
				if !ok {
					return fmt.Errorf("ki.Slice UnmarshalXML: New child of type %v cannot convert to Ki", tn)
				}
				InitNode(kid)
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
