// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/types"
)

// note: use package iox/jsonx for standard read / write of JSON files
// for trees.  The Slice Marshal / Unmarshal methods save the type info
// of each child so that the full tree can be properly reconstructed.

// noMarshalNode is a version of [NodeBase] without a MarshalJSON or UnmarshalJSON
// method (since non-embedded type declarations do not result in method inheritance).
type noMarshalNode NodeBase

// MarshalJSON marshals the node by injecting the [Node.NodeType] as a nodeType
// field at the start of the standard JSON encoding.
func (n *NodeBase) MarshalJSON() ([]byte, error) {
	nmn := (*noMarshalNode)(n)
	b, err := json.Marshal(nmn)
	if err != nil {
		return b, err
	}
	data := `"nodeType":"` + n.This().NodeType().Name + `","numChildren":` + strconv.Itoa(n.NumChildren()) + ","
	b = slices.Insert(b, 1, []byte(data)...)
	return b, nil
}

func (n *NodeBase) UnmarshalJSON(b []byte) error {
	typeStart := bytes.Index(b, []byte(`":"`)) + 3
	typeEnd := bytes.Index(b, []byte(`",`))
	typeName := string(b[typeStart:typeEnd])
	typ := types.TypeByName(typeName)
	if typ == nil {
		return fmt.Errorf("tree.NodeBase.UnmarshalJSON: type %q not found", typeName)
	}

	remainder := b[typeEnd+2:]
	numStart := bytes.Index(remainder, []byte(`":`)) + 2
	numEnd := bytes.Index(remainder, []byte(`,`))
	numString := string(remainder[numStart:numEnd])
	numChildren, err := strconv.Atoi(numString)
	if err != nil {
		return err
	}

	n.DeleteChildren()
	for range numChildren {
		New[*NodeBase](n)
	}

	nmn := (*noMarshalNode)(n)
	return json.Unmarshal(b, nmn)
}

//////////////////////////////////////////////////////////////////////////
// Slice

/*
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
		knm := kid.NodeType().Name
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
			fmt.Println("tree.Slice.MarshalJSON: error doing json.Marshal from kid:", kid)
			errors.Log(err)
			fmt.Println("tree.Slice.MarshalJSON: output to point of error:", string(b))
		}
	}
	b = append(b, []byte("]")...)
	// fmt.Printf("json out: %v\n", string(b))
	return b, nil
}

// UnmarshalJSON parses the length and type information for each object in the
// slice, creates the new slice with those elements, and then loads based on
// the remaining bytes which represent each element
func (sl *Slice) UnmarshalJSON(b []byte) error {
	// fmt.Printf("json in: %v\n", string(b))
	if bytes.Equal(b, []byte("null")) {
		// fmt.Println("\n\n null")
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

	p := make(TypePlan, n)

	for i := 0; i < n; i++ {
		fld := flds[2*i+1]
		// fmt.Printf("fld:\n%v\n", string(fld))
		ti := bytes.Index(fld, []byte("\"type\":"))
		tn := string(bytes.Trim(bytes.TrimSpace(fld[ti+7:]), "\""))
		fld = flds[2*i+2]
		ni := bytes.Index(fld, []byte("\"name\":"))
		nm := string(bytes.Trim(bytes.TrimSpace(fld[ni+7:]), "\""))
		// fmt.Printf("making type: %v\n", tn)
		typ, err := types.TypeByNameTry(tn)
		if err != nil {
			err = fmt.Errorf("tree.Slice UnmarshalJSON: %w", err)
			slog.Error(err.Error())
		}
		p[i].Type = typ
		p[i].Name = nm
	}

	UpdateSlice(sl, nil, p)

	nwk := make([]Node, n) // allocate new slice containing *pointers* to kids
	copy(nwk, *sl)

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
*/

//////////////////////////////////////////////////////
// 	Save / Open Root Type

// The following are special versions for saving the type of
// the root node, which should generally be relatively rare.

// JSONTypePrefix is the first thing output in a tree JSON output file,
// specifying the type of the root node of the tree -- this info appears
// all on one { } bracketed line at the start of the file, and can also be
// used to identify the file as a tree JSON file
var JSONTypePrefix = []byte("{\"tree.RootType\": ")

// JSONTypeSuffix is just the } and \n at the end of the prefix line
var JSONTypeSuffix = []byte("}\n")

// RootTypeJSON returns the JSON encoding of the type of the
// root node (this node) which is written first using our custom
// JSONEncoder type, to enable a file to be loaded de-novo
// and recreate the proper root type for the tree.
func RootTypeJSON(k Node) []byte {
	knm := k.NodeType().Name
	tstr := string(JSONTypePrefix) + fmt.Sprintf("\"%v\"}\n", knm)
	return []byte(tstr)
}

// WriteNewJSON writes JSON-encoded bytes to given writer
// including key type information at start of file
// so ReadNewJSON can create an object of the proper type.
func WriteNewJSON(k Node, writer io.Writer) error {
	tb := RootTypeJSON(k)
	writer.Write(tb)
	return jsonx.WriteIndent(k, writer)
}

// SaveNewJSON writes JSON-encoded bytes to given writer
// including key type information at start of file
// so ReadNewJSON can create an object of the proper type.
func SaveNewJSON(k Node, filename string) error {
	fp, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer fp.Close()
	bw := bufio.NewWriter(fp)
	err = WriteNewJSON(k, bw)
	if err != nil {
		return err
	}
	return bw.Flush()
}

// ReadRootTypeJSON reads the type of the root node
// as encoded by WriteRootTypeJSON, returning the
// types.Type for the saved type name (error if not found),
// the remaining bytes to be decoded using a standard
// unmarshal, and an error.
func ReadRootTypeJSON(b []byte) (*types.Type, []byte, error) {
	if !bytes.HasPrefix(b, JSONTypePrefix) {
		return nil, b, fmt.Errorf("tree.ReadRootTypeJSON -- type prefix not found at start of file -- must be there to identify type of root node of tree")
	}
	stidx := len(JSONTypePrefix) + 1
	eidx := bytes.Index(b, JSONTypeSuffix)
	bodyidx := eidx + len(JSONTypeSuffix)
	tn := string(bytes.Trim(bytes.TrimSpace(b[stidx:eidx]), "\""))
	typ := types.TypeByName(tn)
	if typ == nil {
		return nil, b[bodyidx:], fmt.Errorf("tree.ReadRootTypeJSON: type %q not found", tn)
	}
	return typ, b[bodyidx:], nil
}

// ReadNewJSON reads a new tree from a JSON-encoded byte string,
// using type information at start of file to create an object of the proper type
func ReadNewJSON(reader io.Reader) (Node, error) {
	b, err := io.ReadAll(reader)
	if err != nil {
		return nil, errors.Log(err)
	}
	typ, rb, err := ReadRootTypeJSON(b)
	if err != nil {
		return nil, errors.Log(err)
	}
	root := NewOfType(typ)
	initNode(root)
	err = json.Unmarshal(rb, root)
	UnmarshalPost(root)
	return root, errors.Log(err)
}

// OpenNewJSON opens a new tree from a JSON-encoded file, using type
// information at start of file to create an object of the proper type
func OpenNewJSON(filename string) (Node, error) {
	fp, err := os.Open(filename)
	if err != nil {
		return nil, errors.Log(err)
	}
	defer fp.Close()
	return ReadNewJSON(bufio.NewReader(fp))
}

// ParentAllChildren walks the tree down from current node and call
// SetParent on all children -- needed after an Unmarshal.
func ParentAllChildren(n Node) {
	for _, child := range n.AsTree().Children {
		if child != nil {
			child.AsTree().Par = n
			ParentAllChildren(child)
		}
	}
}

// UnmarshalPost must be called after an Unmarshal;
// calls ParentAllChildren.
func UnmarshalPost(n Node) {
	ParentAllChildren(n)
}
