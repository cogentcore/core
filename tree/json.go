// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/types"
)

// MarshalJSON marshals the node by injecting the [Node.NodeType] as a nodeType
// field and the [NodeBase.NumChildren] as a numChildren field at the start of
// the standard JSON encoding output.
func (n *NodeBase) MarshalJSON() ([]byte, error) {
	// the non pointer value does not implement MarshalJSON, so it will not result in infinite recursion
	b, err := json.Marshal(reflectx.Underlying(reflect.ValueOf(n.This)).Interface())
	if err != nil {
		return b, err
	}
	data := `"nodeType":"` + n.NodeType().Name + `",`
	if n.NumChildren() > 0 {
		data += `"numChildren":` + strconv.Itoa(n.NumChildren()) + ","
	}
	b = slices.Insert(b, 1, []byte(data)...)
	return b, nil
}

// jsonFieldsCache is a cache of the results of [jsonFields] for each type.
var jsonFieldsCache = map[reflect.Type]map[string][]int{}

// jsonFields returns a map from the JSON names of the fields of the given
// struct type to their field indexes, following the encoding/json rules for
// field names and embedded structs. Each name is also included in lowercase,
// to support the case-insensitive matching of encoding/json.
func jsonFields(typ reflect.Type) map[string][]int {
	if fields, ok := jsonFieldsCache[typ]; ok {
		return fields
	}
	fields := map[string][]int{}
	var add func(t reflect.Type, index []int)
	add = func(t reflect.Type, index []int) {
		for i := range t.NumField() {
			f := t.Field(i)
			tag := f.Tag.Get("json")
			if tag == "-" {
				continue
			}
			name, _, _ := strings.Cut(tag, ",")
			fi := append(slices.Clip(index), i)
			if f.Anonymous && name == "" && f.Type.Kind() == reflect.Struct {
				add(f.Type, fi)
				continue
			}
			if !f.IsExported() {
				continue
			}
			if name == "" {
				name = f.Name
			}
			// outer fields take precedence over embedded ones
			if _, has := fields[name]; !has {
				fields[name] = fi
			}
			lower := strings.ToLower(name)
			if _, has := fields[lower]; !has {
				fields[lower] = fi
			}
		}
	}
	add(typ, nil)
	jsonFieldsCache[typ] = fields
	return fields
}

// UnmarshalJSON unmarshals the node by extracting the nodeType field added by
// [NodeBase.MarshalJSON] and then updating the node to the correct type and
// creating children of the correct types. Note that this method can not update the type
// of the node if it has no parent; to load a root node from JSON and have it be of the
// correct type, see the [UnmarshalRootJSON] function. If the type of the node is changed
// by this function, the node pointer will no longer be valid, and the node must be fetched
// again through the children of its parent. You do not need to call [UnmarshalRootJSON]
// or worry about pointers changing if this node is already of the correct type.
func (n *NodeBase) UnmarshalJSON(b []byte) error {
	typ, err := unmarshalNodeType(b)
	if err != nil {
		return err
	}

	// if our type does not match, we must replace our This to make it match
	if n.NodeType() != typ {
		parent := n.Parent
		index := n.IndexInParent()
		if index >= 0 {
			n.Delete()
			n.This = NewOfType(typ)
			parent.AsTree().InsertChild(n.This, index)
			n = n.This.AsTree() // our NodeBase pointer is now different
		}
	}

	// We must delete any existing children first.
	n.DeleteChildren()

	// The children are unmarshaled separately below, so that each one can be
	// made with the correct type before it is unmarshaled, and the fields
	// of this node are unmarshaled without them.
	var fields map[string]json.RawMessage
	err = json.Unmarshal(b, &fields)
	if err != nil {
		return err
	}
	children := fields["Children"]
	delete(fields, "Children")
	delete(fields, "nodeType")
	delete(fields, "numChildren")

	// We unmarshal each field directly, since unmarshaling the node as a
	// whole would call this UnmarshalJSON method again.
	uv := reflectx.Underlying(reflect.ValueOf(n.This))
	jfields := jsonFields(uv.Type())
	for name, data := range fields {
		index, ok := jfields[name]
		if !ok {
			index, ok = jfields[strings.ToLower(name)]
		}
		if !ok {
			continue // unknown fields are ignored, as in encoding/json
		}
		fv, err := uv.FieldByIndexErr(index)
		if err != nil {
			continue // nil embedded pointer
		}
		err = json.Unmarshal(data, fv.Addr().Interface())
		if err != nil {
			return err
		}
	}

	if len(children) == 0 {
		return nil
	}
	var kids []json.RawMessage
	err = json.Unmarshal(children, &kids)
	if err != nil {
		return err
	}
	for _, kb := range kids {
		ktyp, err := unmarshalNodeType(kb)
		if err != nil {
			return err
		}
		kid := NewOfType(ktyp)
		n.AddChild(kid)
		err = kid.AsTree().UnmarshalJSON(kb)
		if err != nil {
			return err
		}
	}
	return nil
}

// unmarshalNodeType returns the type of the node in the given JSON,
// based on the nodeType field added by [NodeBase.MarshalJSON].
func unmarshalNodeType(b []byte) (*types.Type, error) {
	typeName := ""
	key := []byte(`"nodeType"`)
	if i := bytes.Index(b, key); i >= 0 {
		rest := b[i+len(key):]
		if start := bytes.IndexByte(rest, '"'); start >= 0 {
			rest = rest[start+1:]
			if end := bytes.IndexByte(rest, '"'); end >= 0 {
				typeName = string(rest[:end])
			}
		}
	}
	typ := types.TypeByName(typeName)
	if typ == nil {
		return nil, fmt.Errorf("tree.NodeBase.UnmarshalJSON: type %q not found", typeName)
	}
	return typ, nil
}

// UnmarshalRootJSON loads the given JSON to produce a new root node of
// the correct type with all properties and children loaded. If you have
// a root node that you know is already of the correct type, you can just
// call [NodeBase.UnmarshalJSON] on it instead.
func UnmarshalRootJSON(b []byte) (Node, error) {
	// we must make a temporary parent so that the type of the node can be updated
	parent := New[NodeBase]()
	// this NodeBase type is just temporary and will be fixed by [NodeBase.UnmarshalJSON]
	nb := New[NodeBase](parent)
	err := nb.UnmarshalJSON(b)
	if err != nil {
		return nil, err
	}
	// the node must be fetched from the parent's children since the pointer may have changed
	n := parent.Child(0)
	// we must safely remove the node from its temporary parent
	n.AsTree().Parent = nil
	parent.Children = nil
	parent.Destroy()
	return n, nil
}
