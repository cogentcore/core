// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/goki/ki/kit"
)

// Props is the type used for holding generic properties -- the actual Go type
// is a mouthful and not very gui-friendly, and we need some special json methods
type Props map[string]any

var KiT_Props = kit.Types.AddType(&Props{}, PropsProps)

var PropsProps = Props{
	"basic-type": true, // registers props as a basic type avail for type selection in creating property values -- many cases call for nested properties
}

// Set sets props value -- safely creates map
func (pr *Props) Set(key string, val any) {
	if *pr == nil {
		*pr = make(Props)
	}
	(*pr)[key] = val
}

// Prop returns property of given key
func (pr Props) Prop(key string) any {
	return pr[key]
}

// Delete deletes props value at given key
func (pr Props) Delete(key string) {
	delete(pr, key)
}

// SubProps returns a value that contains another props, or nil and false if
// it doesn't exist or isn't a Props
func SubProps(pr map[string]any, key string) (Props, bool) {
	sp, ok := pr[key]
	if !ok {
		return nil, false
	}
	spp, ok := sp.(Props)
	if ok {
		return spp, true
	}
	return nil, false
}

// SubTypeProps returns a value that contains another props, or nil and false if
// it doesn't exist or isn't a Props -- for TypeProps, uses locking
func SubTypeProps(pr map[string]any, key string) (Props, bool) {
	sp, ok := kit.TypeProp(pr, key)
	if !ok {
		return nil, false
	}
	spp, ok := sp.(Props)
	if ok {
		return spp, true
	}
	return nil, false
}

// SetPropStr is a convenience method for e.g., python wrapper that avoids need to deal
// directly with props interface{} type
func SetPropStr(pr Props, key, val string) {
	pr[key] = val
}

// SetSubProps is a convenience method for e.g., python wrapper that avoids need to deal
// directly with props interface{} type
func SetSubProps(pr Props, key string, sp Props) {
	pr[key] = sp
}

// special key prefix indicating type info
var struTypeKey = "__type:"

// special key prefix for enums
var enumTypeKey = "__enum:"

// BlankProp is an empty property, for when there isn't any need for the value
type BlankProp struct{}

// PropStruct is a struct of Name and Value, for use in a PropSlice to hold
// properties that require order information (maps do not retain any order)
type PropStruct struct {
	Name  string
	Value any
}

// PropSlice is a slice of PropStruct, for when order is important within a
// subset of properties (maps do not retain order) -- can set the value of a
// property to a PropSlice to create an ordered list of property values.
type PropSlice []PropStruct

// ElemLabel satisfies the gi.SliceLabeler interface to provide labels for slice elements
func (ps *PropSlice) ElemLabel(idx int) string {
	return (*ps)[idx].Name
}

// SliceProps returns a value that contains a PropSlice, or nil and false if it doesn't
// exist or isn't a PropSlice
func SliceProps(pr map[string]any, key string) (PropSlice, bool) {
	sp, ok := pr[key]
	if !ok {
		return nil, false
	}
	spp, ok := sp.(PropSlice)
	if ok {
		return spp, true
	}
	return nil, false
}

// SliceTypeProps returns a value that contains a PropSlice, or nil and false if it doesn't
// exist or isn't a PropSlice -- for TypeProps, uses locking
func SliceTypeProps(pr map[string]any, key string) (PropSlice, bool) {
	sp, ok := kit.TypeProp(pr, key)
	if !ok {
		return nil, false
	}
	spp, ok := sp.(PropSlice)
	if ok {
		return spp, true
	}
	return nil, false
}

// CopyProps copies properties from source to destination map.  If deepCopy
// is true, then any values that are Props or PropSlice are copied too
// *dest can be nil, in which case it is created.
func CopyProps(dest *map[string]any, src map[string]any, deepCopy bool) {
	if *dest == nil {
		*dest = make(Props, len(src))
	}
	for key, val := range src {
		if deepCopy {
			if pv, ok := val.(map[string]any); ok {
				var nval Props
				nval.CopyFrom(pv, deepCopy)
				(*dest)[key] = nval
				continue
			} else if pv, ok := val.(Props); ok {
				var nval Props
				nval.CopyFrom(pv, deepCopy)
				(*dest)[key] = nval
				continue
			} else if pv, ok := val.(PropSlice); ok {
				var nval PropSlice
				nval.CopyFrom(pv, deepCopy)
				(*dest)[key] = nval
				continue
			}
		}
		(*dest)[key] = val
	}
}

// CopyFrom copies properties from source to receiver destination map.  If deepCopy
// is true, then any values that are Props or PropSlice are copied too
// *dest can be nil, in which case it is created.
func (dest *Props) CopyFrom(src map[string]any, deepCopy bool) {
	CopyProps((*map[string]any)(dest), src, deepCopy)
}

// CopyFrom copies properties from source to destination propslice.  If deepCopy
// is true, then any values that are Props or PropSlice are copied too
// *dest can be nil, in which case it is created.
func (dest *PropSlice) CopyFrom(src PropSlice, deepCopy bool) {
	if *dest == nil {
		*dest = make(PropSlice, len(src))
	}
	for i, val := range src {
		if deepCopy {
			if pv, ok := val.Value.(map[string]any); ok {
				var nval Props
				CopyProps((*map[string]any)(&nval), pv, deepCopy)
				(*dest)[i] = PropStruct{Name: val.Name, Value: nval}
				continue
			} else if pv, ok := val.Value.(Props); ok {
				var nval Props
				CopyProps((*map[string]any)(&nval), pv, deepCopy)
				(*dest)[i] = PropStruct{Name: val.Name, Value: nval}
				continue
			} else if pv, ok := val.Value.(PropSlice); ok {
				var nval PropSlice
				nval.CopyFrom(pv, deepCopy)
				(*dest)[i] = PropStruct{Name: val.Name, Value: nval}
				continue
			}
		}
		(*dest)[i] = src[i]
	}
}

// MarshalJSON saves the type information for each struct used in props, as a
// separate key with the __type: prefix -- this allows the Unmarshal to
// create actual types
func (p Props) MarshalJSON() ([]byte, error) {
	nk := len(p)
	b := make([]byte, 0, nk*100+20)
	if nk == 0 {
		b = append(b, []byte("null")...)
		return b, nil
	}
	b = append(b, []byte("{")...)
	cnt := 0
	var err error
	for key, val := range p {
		vt := kit.NonPtrType(reflect.TypeOf(val))
		vk := vt.Kind()
		if vk == reflect.Struct {
			knm := kit.Types.TypeName(vt)
			tstr := fmt.Sprintf("\"%v%v\": \"%v\",", struTypeKey, key, knm)
			b = append(b, []byte(tstr)...)
		}
		kstr := fmt.Sprintf("\"%v\": ", key)
		b = append(b, []byte(kstr)...)

		var kb []byte
		kb, err = json.Marshal(val)
		if err != nil {
			log.Printf("error doing json.Marshall from val: %v\n%v\n", val, err)
			log.Printf("output to point of error: %v\n", string(b))
		} else {
			if vk >= reflect.Int && vk <= reflect.Uint64 && kit.Enums.TypeRegistered(vt) {
				knm := kit.Types.TypeName(vt)
				kb, _ = json.Marshal(val)
				estr := fmt.Sprintf("\"%v(%v)%v\"", enumTypeKey, knm, string(bytes.Trim(kb, "\"")))
				b = append(b, []byte(estr)...)
			} else {
				b = append(b, kb...)
			}
		}
		if cnt < nk-1 {
			b = append(b, []byte(",")...)
		}
		cnt++
	}
	b = append(b, []byte("}")...)
	// fmt.Printf("json out: %v\n", string(b))
	return b, nil
}

// UnmarshalJSON parses the type information in the map to restore actual
// objects -- this is super inefficient and really needs a native parser, but
// props are likely to be relatively small
func (p *Props) UnmarshalJSON(b []byte) error {
	// fmt.Printf("json in: %v\n", string(b))
	if bytes.Equal(b, []byte("null")) {
		*p = nil
		return nil
	}

	// load into a temporary map and then process
	tmp := make(map[string]any)
	err := json.Unmarshal(b, &tmp)
	if err != nil {
		return err
	}

	*p = make(Props, len(tmp))

	// create all the structure objects from the list -- have to do this first to get all
	// the structs made b/c the order is random..
	for key, val := range tmp {
		if strings.HasPrefix(key, struTypeKey) {
			pkey := strings.TrimPrefix(key, struTypeKey)
			rval := tmp[pkey]
			tn := val.(string)
			typ := kit.Types.Type(tn)
			if typ == nil {
				log.Printf("ki.Props: cannot load struct of type %v -- not registered in kit.Types\n", tn)
				continue
			}
			if IsKi(typ) { // note: not really a good idea to store ki's in maps, but..
				kival := NewOfType(typ)
				InitNode(kival)
				if kival != nil {
					// fmt.Printf("stored new ki of type %v in key: %v\n", typ.String(), pkey)
					tmpb, _ := json.Marshal(rval) // string rep of this
					err = kival.ReadJSON(bytes.NewReader(tmpb))
					if err != nil {
						log.Printf("ki.Props failed to load Ki struct of type %v with error: %v\n", typ.String(), err)
					}
					(*p)[pkey] = kival
				}
			} else {
				stval := reflect.New(typ).Interface()
				// fmt.Printf("stored new struct of type %v in key: %v\n", typ.String(), pkey)
				tmpb, _ := json.Marshal(rval) // string rep of this
				err = json.Unmarshal(tmpb, stval)
				if err != nil {
					log.Printf("ki.Props failed to load struct of type %v with error: %v\n", typ.String(), err)
				}
				(*p)[pkey] = reflect.ValueOf(stval).Elem().Interface()
			}
		}
	}

	// now can re-iterate
	for key, val := range tmp {
		if strings.HasPrefix(key, struTypeKey) {
			continue
		}
		if _, ok := (*p)[key]; ok { // already created -- was a struct -- skip
			continue
		}
		// look for sub-maps, make them props..
		if _, ok := val.(map[string]any); ok {
			// fmt.Printf("stored new Props map in key: %v\n", key)
			subp := Props{}
			tmpb, _ := json.Marshal(val) // string rep of this
			err = json.Unmarshal(tmpb, &subp)
			if err != nil {
				log.Printf("ki.Props failed to load sub-Props with error: %v\n", err)
			}
			(*p)[key] = subp
		} else { // straight copy
			if sval, ok := val.(string); ok {
				if strings.HasPrefix(sval, enumTypeKey) {
					tn := strings.TrimPrefix(sval, enumTypeKey)
					rpi := strings.Index(tn, ")")
					str := tn[rpi+1:]
					tn = tn[1:rpi]
					etyp := kit.Enums.Enum(tn)
					if etyp != nil {
						eval := kit.EnumIfaceFromString(str, etyp)
						(*p)[key] = eval
						// fmt.Printf("decoded enum typ %v into actual value: %v from %v\n", etyp.String(), eval, str)
					} else {
						(*p)[key] = str
					}
					continue
				}
			}
			(*p)[key] = val
		}
	}

	return nil
}
