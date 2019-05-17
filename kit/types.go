// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package kit provides various reflect type functions for GoKi system, including:
//
// * kit.TypeRegistry (types.go) for associating string names with
// reflect.Type values, to allow dynamic marshaling of structs, and also
// bidirectional string conversion of const int iota (enum) types.  It is used
// by the GoKi ki system, hence the kit (ki types) name.
//
// To register a new type, add:
//
// var KiT_TypeName = kit.Types.AddType(&TypeName{}, [props|nil])
//
// where the props is a map[string]interface{} of optional properties that can
// be associated with the type -- this is used in the GoGi graphical interface
// system for example to color objects of different types using the
// background-color property.  KiT_TypeName variable can be conveniently used
// wherever a reflect.Type of that type is needed.
//
// * kit.EnumRegistry (enums.go) that registers constant int iota (aka enum) types, and
// provides general conversion utilities to / from string, int64, general
// properties associated with enum types, and deals with bit flags
//
// * kit.Type (type.go) struct provides JSON and XML Marshal / Unmarshal functions for
// saving / loading reflect.Type using registrered type names.
//
// * convert.go: robust interface{}-based type conversion routines that are
// useful in more lax user-interface contexts where "common sense" conversions
// between strings, numbers etc are useful
//
// * embeds.go: various functions for managing embedded struct types, e.g.,
// determining if a given type embeds another type (directly or indirectly),
// and iterating over fields to flatten the otherwise nested nature of the
// field encoding in embedded types.
package kit

import (
	"log"
	"path"
	"reflect"
	"sync"
)

// TypeRegistry contains several maps where properties of types
// are maintained for general usage.
//
// See also EnumRegistry for a version specifically for "enum" types
// (const int types that associate a name with an int value).
//
// Each type must be explicitly registered by calling AddType
// in an expression that also initializes a new global variable
// that is then useful whenever you need to specify that type,
// e.g., when adding a new Node in the Ki system.
//
// var KiT_MyType = ki.Types.AddType(&MyType{}, [props|nil])
//
// where MyType is the type -- note that it is ESSENTIAL to pass a pointer
// so that the type is considered addressable, even after we get Elem() of it.
//
// props is a map[string]interface{} of optional properties that can be
// associated with the type -- this is used in the GoGi graphical interface
// system for example to configure menus and toolbars for different types.
//
// Once registered, the reflect.Type can be looked up via the usual
// *short* package-qualified type name that you use in programming
// (e.g., kit.TypeRegistry) -- this is maintained in the Types map.
// This lookup enables e.g., saving type name information in JSON files
// and then using that type name to create an object of that type
// upon loading.
//
// The Props map stores the properties (which can be updated during
// runtime as well), and Insts also stores an interface{} pointer to
// each type that has been registered.
type TypeRegistry struct {
	// Types is a map from the *short* package qualified name to reflect.Type
	Types map[string]reflect.Type

	// ShortNames is a map of short package qualified names keyed by
	// the long, fully unambiguous package path qualified name.
	// It is somewhat expensive to compute this short name, so
	// caching it is faster
	ShortNames map[string]string

	// Props are type properties -- nodes can get default properties from
	// their types and then optionally override them with their own settings.
	// The key here is the long, full type name.
	Props map[string]map[string]interface{}

	// Insts contain an instance of each type (the one passed during AddType)
	// The key here is the long, full type name.
	Insts map[string]interface{}
}

// Types is master registry of types that embed Ki Nodes
var Types TypeRegistry

// LongTypeName returns the long, full package-path qualified type name.
// This is guaranteed to be unique and used for internal storage of
// several maps to avoid any conflicts.  It is also very quick to compute.
func LongTypeName(typ reflect.Type) string {
	return typ.PkgPath() + "." + typ.Name()
}

// ShortTypeName returns the short version of a package-qualified type name
// which just has the last element of the path.  This is what is used in
// standard Go programming, and is is used for the key to lookup reflect.Type
// names -- i.e., this is what you should save in a JSON file.
// The potential naming conflict is worth the brevity, and typically a given
// file will only contain mutually-compatible, non-conflicting types.
// This is cached in ShortNames because the path.Base computation is apparently
// a bit slow.
func ShortTypeName(typ reflect.Type) string {
	return path.Base(typ.PkgPath()) + "." + typ.Name()
}

// TypesMu protects updating of the type registry maps -- main Addtype etc all
// happens at startup and does not need protection, but property access does.
// use RLock for read-access to properties, and Lock for write access when
// adding or changing key / value.
var TypesMu sync.RWMutex

// AddType adds a given type to the registry -- requires an empty object to
// grab type info from (which is then stored in Insts) -- must be passed as a
// pointer to ensure that it is an addressable, settable type -- also optional
// properties that can be associated with the type and accessible e.g. for
// view-specific properties etc -- these props MUST be specific to this type
// as they are used directly, not copied!!
func (tr *TypeRegistry) AddType(obj interface{}, props map[string]interface{}) reflect.Type {
	if tr.Types == nil {
		tr.Init()
	}

	typ := reflect.TypeOf(obj).Elem()
	lnm := LongTypeName(typ)
	snm := ShortTypeName(typ)
	tr.ShortNames[lnm] = snm
	tr.Types[snm] = typ
	tr.Insts[lnm] = obj
	if props != nil {
		tr.Props[lnm] = props
	}
	return typ
}

// TypeName returns the *short* package-qualified type name for given reflect.Type.
// This is the version that should be used in saving the type to a file, etc.
// It uses a map for fast results.
func (tr *TypeRegistry) TypeName(typ reflect.Type) string {
	lnm := LongTypeName(typ)
	if snm, ok := tr.ShortNames[lnm]; ok {
		return snm
	}
	snm := ShortTypeName(typ)
	TypesMu.Lock()
	tr.ShortNames[lnm] = snm
	TypesMu.Unlock()
	return snm
}

// Type returns the reflect.Type based on its *short* package-qualified name
// (package directory + "." + type -- the version that you use in programming).
// Returns nil if not registered.
func (tr *TypeRegistry) Type(typeName string) reflect.Type {
	if typ, ok := tr.Types[typeName]; ok {
		return typ
	}
	return nil
}

// InstByName returns the interface{} instance of given type (it is a pointer
// to that type) using the long, unambiguous package-qualified name.
// Returns nil if not found.
func (tr *TypeRegistry) InstByName(typeName string) interface{} {
	if inst, ok := tr.Insts[typeName]; ok {
		return inst
	}
	return nil
}

// Inst returns the interface{} instance of given type (it is a pointer
// to that type).  Returns nil if not found.
func (tr *TypeRegistry) Inst(typ reflect.Type) interface{} {
	return tr.InstByName(LongTypeName(typ))
}

// PropsByName returns properties for given type name, using the long,
// unambiguous package-qualified name.
// It optionally makes props map for this type if not already made.
// Can use this to register properties for types that are not registered.
func (tr *TypeRegistry) PropsByName(typeName string, makeNew bool) *map[string]interface{} {
	TypesMu.Lock()
	defer TypesMu.Unlock()
	tp, ok := tr.Props[typeName]
	if !ok {
		if !makeNew {
			return nil
		}
		tp = make(map[string]interface{})
		tr.Props[typeName] = tp
	}
	return &tp
}

// Properties returns properties for given type.
// It optionally makes props map for this type if not already made.
// Can use this to register properties for types that are not registered.
func (tr *TypeRegistry) Properties(typ reflect.Type, makeNew bool) *map[string]interface{} {
	return tr.PropsByName(LongTypeName(typ), makeNew)
}

// TypeProp provides safe (mutex protected) read access to property map
// returned by Properties method -- must use this for all Properties access!
func TypeProp(props map[string]interface{}, key string) (interface{}, bool) {
	TypesMu.RLock()
	val, ok := props[key]
	TypesMu.RUnlock()
	return val, ok
}

// SetTypeProp provides safe (mutex protected) write setting of property map
// returned by Properties method -- must use this for all Properties access!
func SetTypeProp(props map[string]interface{}, key string, val interface{}) {
	TypesMu.Lock()
	props[key] = val
	TypesMu.Unlock()
}

// PropByName safely finds a type property from type name (using the long,
// unambiguous package-qualified name) and property key.
// Returns false if not found
func (tr *TypeRegistry) PropByName(typeName, propKey string) (interface{}, bool) {
	TypesMu.RLock()
	defer TypesMu.RUnlock()

	tp, ok := tr.Props[typeName]
	if !ok {
		// fmt.Printf("no props for type: %v\n", typeName)
		return nil, false
	}
	p, ok := tp[propKey]
	return p, ok
}

// Prop safely finds a type property from type and property key -- returns
// false if not found.
func (tr *TypeRegistry) Prop(typ reflect.Type, propKey string) (interface{}, bool) {
	return tr.PropByName(LongTypeName(typ), propKey)
}

// SetProps sets the type props for given type, uses write mutex lock
func (tr *TypeRegistry) SetProps(typ reflect.Type, props map[string]interface{}) {
	TypesMu.Lock()
	defer TypesMu.Unlock()
	tr.Props[LongTypeName(typ)] = props
}

// AllImplementersOf returns a list of all registered types that implement the
// given interface type at any level of embedding -- must pass a type
// constructed like this: reflect.TypeOf((*gi.Node2D)(nil)).Elem() --
// includeBases indicates whether to include types marked with property of
// base-type -- typically not useful for user-facing type selection
func (tr *TypeRegistry) AllImplementersOf(iface reflect.Type, includeBases bool) []reflect.Type {
	if iface.Kind() != reflect.Interface {
		log.Printf("kit.TypeRegistry AllImplementersOf -- type is not an interface: %v\n", iface)
		return nil
	}
	tl := make([]reflect.Type, 0)
	for _, typ := range tr.Types {
		if !includeBases {
			if btp, ok := tr.Prop(typ, "base-type"); ok {
				if bt, ok := ToBool(btp); ok && bt {
					continue
				}
			}
		}
		nptyp := NonPtrType(typ)
		if nptyp.Kind() != reflect.Struct {
			continue
		}
		if EmbedImplements(typ, iface) {
			tl = append(tl, typ)
		}
	}
	return tl
}

// AllEmbedsOf returns a list of all registered types that embed (inherit from
// in C++ terminology) the given type -- inclusive determines whether the type
// itself is included in list -- includeBases indicates whether to include
// types marked with property of base-type -- typically not useful for
// user-facing type selection
func (tr *TypeRegistry) AllEmbedsOf(embed reflect.Type, inclusive, includeBases bool) []reflect.Type {
	tl := make([]reflect.Type, 0)
	for _, typ := range tr.Types {
		if !inclusive && typ == embed {
			continue
		}
		if !includeBases {
			if btp, ok := tr.Prop(typ, "base-type"); ok {
				if bt, ok := ToBool(btp); ok && bt {
					continue
				}
			}
		}
		if TypeEmbeds(typ, embed) {
			tl = append(tl, typ)
		}
	}
	return tl
}

// AllTagged returns a list of all registered types that include a given
// property key value -- does not check for the value of that value -- just
// its existence
func (tr *TypeRegistry) AllTagged(key string) []reflect.Type {
	tl := make([]reflect.Type, 0)
	for _, typ := range tr.Types {
		_, ok := tr.Prop(typ, key)
		if !ok {
			continue
		}
		tl = append(tl, typ)
	}
	return tl
}

// Init initializes the type registry, including adding basic types
func (tr *TypeRegistry) Init() {
	tr.Types = make(map[string]reflect.Type, 1000)
	tr.Insts = make(map[string]interface{}, 1000)
	tr.Props = make(map[string]map[string]interface{}, 1000)
	tr.ShortNames = make(map[string]string, 1000)

	{
		var BoolProps = map[string]interface{}{
			"basic-type": true,
		}
		ob := false
		tr.AddType(&ob, BoolProps)
	}
	{
		var IntProps = map[string]interface{}{
			"basic-type": true,
		}
		ob := int(0)
		tr.AddType(&ob, IntProps)
	}
	{
		ob := int8(0)
		tr.AddType(&ob, nil)
	}
	{
		ob := int16(0)
		tr.AddType(&ob, nil)
	}
	{
		ob := int32(0)
		tr.AddType(&ob, nil)
	}
	{
		ob := int64(0)
		tr.AddType(&ob, nil)
	}
	{
		ob := uint(0)
		tr.AddType(&ob, nil)
	}
	{
		ob := uint8(0)
		tr.AddType(&ob, nil)
	}
	{
		ob := uint16(0)
		tr.AddType(&ob, nil)
	}
	{
		ob := uint32(0)
		tr.AddType(&ob, nil)
	}
	{
		ob := uint64(0)
		tr.AddType(&ob, nil)
	}
	{
		ob := uintptr(0)
		tr.AddType(&ob, nil)
	}
	{
		ob := float32(0)
		tr.AddType(&ob, nil)
	}
	{
		var Float64Props = map[string]interface{}{
			"basic-type": true,
		}
		ob := float64(0)
		tr.AddType(&ob, Float64Props)
	}
	{
		ob := complex64(0)
		tr.AddType(&ob, nil)
	}
	{
		ob := complex128(0)
		tr.AddType(&ob, nil)
	}
	{
		var StringProps = map[string]interface{}{
			"basic-type": true,
		}
		ob := string(0)
		tr.AddType(&ob, StringProps)
	}
}
