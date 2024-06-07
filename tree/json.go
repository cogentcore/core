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
	"reflect"
	"slices"
	"strconv"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/types"
)

// MarshalJSON marshals the node by injecting the [Node.NodeType] as a nodeType
// field and the [NodeBase.NumChildren] as a numChildren field at the start of
// the standard JSON encoding output.
func (n *NodeBase) MarshalJSON() ([]byte, error) {
	// the non pointer value does not implement MarshalJSON, so it will not result in infinite recursion
	b, err := json.Marshal(reflectx.Underlying(reflect.ValueOf(n.Ths)).Interface())
	if err != nil {
		return b, err
	}
	data := `"nodeType":"` + n.This().NodeType().Name + `","numChildren":` + strconv.Itoa(n.NumChildren()) + ","
	b = slices.Insert(b, 1, []byte(data)...)
	return b, nil
}

// unmarshalTypeCache is a cache of [reflect.Type] values used
// for unmarshalling in [NodeBase.UnmarshalJSON]. This cache has
// a noticeable performance benefit of around 1.2x in
// [BenchmarkNodeUnmarshalJSON], a benefit that should only increase
// for larger trees.
var unmarshalTypeCache = map[string]reflect.Type{}

// UnmarshalJSON unmarshals the node by extracting the nodeType and numChildren fields
// added by [NodeBase.MarshalJSON] and then updating the node to the correct type and
// creating the correct number of children. Note that this method can not update the type
// of the node if it has no parent; to load a root node from JSON and have it be of the
// correct type, see the [UnmarshalRootJSON] function. If the type of the node is changed
// by this function, the node pointer will no longer be valid, and the node must be fetched
// again through the children of its parent.
func (n *NodeBase) UnmarshalJSON(b []byte) error {
	typeStart := bytes.Index(b, []byte(`":`)) + 3
	typeEnd := bytes.Index(b, []byte(`",`))
	typeName := string(b[typeStart:typeEnd])
	// we may end up with an extraneous quote / space at the start
	typeName = strings.TrimPrefix(strings.TrimSpace(typeName), `"`)
	typ := types.TypeByName(typeName)
	if typ == nil {
		return fmt.Errorf("tree.NodeBase.UnmarshalJSON: type %q not found", typeName)
	}

	// if our type does not match, we must replace our This to make it match
	if n.Ths.NodeType() != typ {
		parent := n.Par
		index := n.IndexInParent()
		if index >= 0 {
			n.Delete()
			n.Ths = parent.AsTree().InsertNewChild(typ, index)
			n = n.Ths.AsTree() // our NodeBase pointer is now different
		}
	}

	remainder := b[typeEnd+2:]
	numStart := bytes.Index(remainder, []byte(`":`)) + 2
	numEnd := bytes.Index(remainder, []byte(`,`))
	numString := string(remainder[numStart:numEnd])
	// we may end up with extraneous space at the start
	numString = strings.TrimSpace(numString)
	numChildren, err := strconv.Atoi(numString)
	if err != nil {
		return err
	}

	// We delete any existing children and then make placeholder NodeBase children
	// that will be replaced with children of the correct type during their UnmarshalJSON.
	n.DeleteChildren()
	for range numChildren {
		New[*NodeBase](n)
	}

	uv := reflectx.UnderlyingPointer(reflect.ValueOf(n.Ths))
	rtyp := unmarshalTypeCache[typeName]
	if rtyp == nil {
		// We must create a new type that has the exact same fields as the original type
		// so that we can unmarshal into it without having infinite recursion on the
		// UnmarshalJSON method. This works because [reflect.StructOf] does not promote
		// methods on embedded fields, meaning that the UnmarshalJSON method on the NodeBase
		// is not carried over and thus is not called, avoiding infinite recursion.
		uvt := uv.Type().Elem()
		fields := make([]reflect.StructField, uvt.NumField())
		for i := range fields {
			fields[i] = uvt.Field(i)
		}
		nt := reflect.StructOf(fields)
		rtyp = reflect.PointerTo(nt)
		unmarshalTypeCache[typeName] = rtyp
	}
	// We can directly convert because our new struct type has the exact same fields.
	uvi := uv.Convert(rtyp).Interface()
	err = json.Unmarshal(b, uvi)
	if err != nil {
		return err
	}
	return nil
}

// UnmarshalRootJSON loads the given JSON to produce a new root node of
// the correct type with all properties and children loaded.
func UnmarshalRootJSON(b []byte) (Node, error) {
	// we must make a temporary parent so that the type of the node can be updated
	parent := New[*NodeBase]()
	// this NodeBase type is just temporary and will be fixed by [NodeBase.UnmarshalJSON]
	nb := New[*NodeBase](parent)
	err := nb.UnmarshalJSON(b)
	if err != nil {
		return nil, err
	}
	// the node must be fetched from the parent's children since the pointer may have changed
	n := parent.Child(0)
	// we must safely remove the node from its temporary parent
	n.AsTree().Par = nil
	parent.Children = nil
	parent.Destroy()
	return n, nil
}

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
