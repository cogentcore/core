// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// note: parsing code adapted from pflag package https://github.com/spf13/pflag
// Copyright (c) 2012 Alex Ogier. All rights reserved.
// Copyright (c) 2012 The Go Authors. All rights reserved.

package grease

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/iancoleman/strcase"
	"goki.dev/ki/v2/kit"
	"goki.dev/ki/v2/toml"
)

// SetFromArgs sets Config values from command-line args,
// based on the field names in the Config struct.
// Returns any args that did not start with a `-` flag indicator.
// For more robust error processing, it is assumed that all flagged args (-)
// must refer to fields in the config, so any that fail to match trigger
// an error.  Errors can also result from parsing.
// Errors are automatically logged because these are user-facing.
func SetFromArgs(cfg any, args []string) (nonFlags []string, err error) {
	allArgs := make(map[string]reflect.Value)
	CommandArgs(allArgs) // need these to not trigger not-found errors
	FieldArgNames(cfg, allArgs)
	nonFlags, err = ParseArgs(cfg, args, allArgs, true)
	if err != nil {
		fmt.Println(Usage(cfg))
	}
	return
}

// ParseArgs parses given args using map of all available args
// setting the value accordingly, and returning any leftover args.
// setting errNotFound = true causes args that are not in allArgs to
// trigger an error.  Otherwise, it just skips those.
func ParseArgs(cfg any, args []string, allArgs map[string]reflect.Value, errNotFound bool) ([]string, error) {
	var nonFlags []string
	var err error
	for len(args) > 0 {
		s := args[0]
		args = args[1:]
		if len(s) == 0 || s[0] != '-' || len(s) == 1 {
			nonFlags = append(nonFlags, s)
			continue
		}

		if s[1] == '-' && len(s) == 2 { // "--" terminates the flags
			// f.argsLenAtDash = len(f.args)
			nonFlags = append(nonFlags, args...)
			break
		}
		args, err = ParseArg(s, args, allArgs, errNotFound)
		if err != nil {
			return nonFlags, err
		}
	}
	return nonFlags, nil
}

func ParseArg(s string, args []string, allArgs map[string]reflect.Value, errNotFound bool) (a []string, err error) {
	a = args
	name := s[1:]
	if name[0] == '-' {
		name = name[1:]
	}
	if len(name) == 0 || name[0] == '-' || name[0] == '=' {
		err = fmt.Errorf("grease.ParseArgs: bad flag syntax: %s", s)
		fmt.Println(err)
		return
	}

	if strings.HasPrefix(name, "test.") { // go test passes args..
		return
	}

	split := strings.SplitN(name, "=", 2)
	name = split[0]
	fval, exists := allArgs[name]
	if !exists {
		if errNotFound {
			err = fmt.Errorf("grease.ParseArgs: flag name not recognized: %s", name)
			fmt.Println(err)
		}
		return
	}

	isbool := kit.NonPtrValue(fval).Kind() == reflect.Bool

	var value string
	switch {
	case len(split) == 2:
		// '--flag=arg'
		value = split[1]
	case isbool:
		// '--flag' bare
		lcnm := strings.ToLower(name)
		negate := false
		if len(lcnm) > 3 {
			if lcnm[:3] == "no_" || lcnm[:3] == "no-" {
				negate = true
			} else if lcnm[:2] == "no" {
				if _, has := allArgs[lcnm[2:]]; has { // e.g., nogui and gui is on list
					negate = true
				}
			}
		}
		if negate {
			value = "false"
		} else {
			value = "true"
		}
	case len(a) > 0:
		// '--flag arg'
		value = a[0]
		a = a[1:]
	default:
		// '--flag' (arg was required)
		err = fmt.Errorf("grease.ParseArgs: flag needs an argument: %s", s)
		fmt.Println(err)
		return
	}

	err = SetArgValue(name, fval, value)
	return
}

// SetArgValue sets given arg name to given value, into settable reflect.Value
func SetArgValue(name string, fval reflect.Value, value string) error {
	nptyp := kit.NonPtrType(fval.Type())
	vk := nptyp.Kind()
	switch {
	case vk >= reflect.Int && vk <= reflect.Uint64 && kit.Enums.TypeRegistered(nptyp):
		return kit.Enums.SetAnyEnumValueFromString(fval, value)
	case vk == reflect.Map:
		mval := make(map[string]any)
		err := toml.ReadBytes(&mval, []byte("tmp="+value)) // use toml decoder
		if err != nil {
			fmt.Println(err)
			return err
		}
		err = kit.CopyMapRobust(fval.Interface(), mval["tmp"])
		if err != nil {
			fmt.Println(err)
			err = fmt.Errorf("grease.ParseArgs: not able to set map field from arg: %s val: %s", name, value)
			fmt.Println(err)
			return err
		}
	case vk == reflect.Slice:
		mval := make(map[string]any)
		err := toml.ReadBytes(&mval, []byte("tmp="+value)) // use toml decoder
		if err != nil {
			fmt.Println(err)
			return err
		}
		err = kit.CopySliceRobust(fval.Interface(), mval["tmp"])
		if err != nil {
			fmt.Println(err)
			err = fmt.Errorf("grease.ParseArgs: not able to set slice field from arg: %s val: %s", name, value)
			fmt.Println(err)
			return err
		}
	default:
		ok := kit.SetRobust(fval.Interface(), value) // overkill but whatever
		if !ok {
			err := fmt.Errorf("grease.ParseArgs: not able to set field from arg: %s val: %s", name, value)
			fmt.Println(err)
			return err
		}
	}
	return nil
}

// FieldArgNames adds to given args map all the different ways the field names
// can be specified as arg flags, mapping to the reflect.Value
func FieldArgNames(obj any, allArgs map[string]reflect.Value) {
	fieldArgNamesStruct(obj, "", false, allArgs)
}

func addAllCases(nm, path string, pval reflect.Value, allArgs map[string]reflect.Value) {
	if nm == "Includes" {
		return // skip
	}
	if path != "" {
		nm = path + "." + nm
	}
	allArgs[nm] = pval
	allArgs[strings.ToLower(nm)] = pval
	allArgs[strcase.ToKebab(nm)] = pval
	allArgs[strcase.ToSnake(nm)] = pval
	allArgs[strcase.ToScreamingSnake(nm)] = pval
}

// fieldArgNamesStruct returns map of all the different ways the field names
// can be specified as arg flags, mapping to the reflect.Value
func fieldArgNamesStruct(obj any, path string, nest bool, allArgs map[string]reflect.Value) {
	if kit.IfaceIsNil(obj) {
		return
	}
	ov := reflect.ValueOf(obj)
	if ov.Kind() == reflect.Pointer && ov.IsNil() {
		return
	}
	val := kit.NonPtrValue(ov)
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		fv := val.Field(i)
		if kit.NonPtrType(f.Type).Kind() == reflect.Struct {
			nwPath := f.Name
			if path != "" {
				nwPath = path + "." + nwPath
			}
			nwNest := nest
			if !nwNest {
				neststr, ok := f.Tag.Lookup("nest")
				if ok && (neststr == "+" || neststr == "true") {
					nwNest = true
				}
			}
			fieldArgNamesStruct(kit.PtrValue(fv).Interface(), nwPath, nwNest, allArgs)
			continue
		}
		pval := kit.PtrValue(fv)
		addAllCases(f.Name, path, pval, allArgs)
		if f.Type.Kind() == reflect.Bool {
			addAllCases("No"+f.Name, path, pval, allArgs)
		}
		// now process adding non-nested version of field
		if path == "" || nest {
			continue
		}
		neststr, ok := f.Tag.Lookup("nest")
		if ok && (neststr == "+" || neststr == "true") {
			continue
		}
		if _, has := allArgs[f.Name]; has {
			fmt.Printf("warning: programmer error: grease config field \"%s.%s\" cannot be added as a non-nested flag with the name %q because that name has already been registered by another field; add the field tag 'nest:\"+\"' to the field you want to require nested access for (ie: \"Path.Field\" instead of \"Field\") to remove this warning\n", path, f.Name, f.Name)
			continue
		}
		addAllCases(f.Name, "", pval, allArgs)
		if f.Type.Kind() == reflect.Bool {
			addAllCases("No"+f.Name, "", pval, allArgs)
		}
	}
}

// CommandArgs adds non-field args that control the config process:
// -config -cfg -help -h
func CommandArgs(allArgs map[string]reflect.Value) {
	allArgs["config"] = reflect.ValueOf(&ConfigFile)
	allArgs["cfg"] = reflect.ValueOf(&ConfigFile)
	allArgs["help"] = reflect.ValueOf(&Help)
	allArgs["h"] = reflect.ValueOf(&Help)
}
