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
	nonFlags, err = ParseFlags(cfg, args, allArgs, true)
	return
}

// GetArgs processes the given args using map of all available args,
// returning the leftover (positional) args, the flags, and any error.
// setting errNotFound = true causes args that are not in allArgs to
// trigger an error.  Otherwise, it just skips those.
func GetArgs(args []string) ([]string, map[string]string, error) {
	var nonFlags []string
	flags := map[string]string{}
	for len(args) > 0 {
		s := args[0]
		args = args[1:]
		if len(s) == 0 || s[0] != '-' || len(s) == 1 { // if we are not a flag, just add to non-flags
			nonFlags = append(nonFlags, s)
			continue
		}

		if s[1] == '-' && len(s) == 2 { // "--" terminates the flags
			// f.argsLenAtDash = len(f.args)
			nonFlags = append(nonFlags, args...)
			break
		}
		name, value, nargs, err := GetFlag(s, args)
		if err != nil {
			return nonFlags, flags, err
		}
		// we need to updated remaining args with latest
		args = nargs
		flags[name] = value
	}
	return nonFlags, flags, nil
}

// GetFlag parses the given flag arg string in the context
// of the given remaining arguments, and returns the
// name of the flag, the value of the flag, the remaining
// arguments updated with any changes caused by getting
// this flag, and any error. It is designed for use in [GetArgs]
// and should typically not be used by end-user code.
func GetFlag(s string, args []string) (name, value string, a []string, err error) {
	// we start out with the remaining args we were passed
	a = args
	// we know the first character is a dash, so we can trim it directly
	name = s[1:]
	// then we trim double dash if there is one
	name = strings.TrimPrefix(name, "-")

	// we can't start with a dash or equal, as those are reserved characters
	if len(name) == 0 || name[0] == '-' || name[0] == '=' {
		err = fmt.Errorf("bad flag syntax: %q", s)
		return
	}

	// go test passes args, so we ignore them
	if strings.HasPrefix(name, "test.") {
		return
	}

	// split on equal (we could be in the form flag=value)
	split := strings.SplitN(name, "=", 2)
	name = split[0]
	if len(split) == 2 {
		// if we are in the form flag=value, we are done
		value = split[1]
	} else if len(a) > 0 { // otherwise, if we still have more remaining args, our value could be the next arg (if we have no remaining args, we are a terminating bool arg)
		value = a[0]
		// if the next arg starts with a dash, it can't be our value, so we are just a bool arg and we exit with an empty value
		if strings.HasPrefix(value, "-") {
			value = ""
			return
		} else {
			// if it doesn't start with a dash, it is our value, so we remove it from the remaining args (we have already set value to it above)
			a = a[1:]
			return
		}
	}
	return
}

// ParseArgs parses the given non-flag arguments in the context of the given
// configuration information and commands. The non-flag arguments should be
// gotten through [GetArgs] first.
func ParseArgs[T any](cfg T, args []string, cmds ...*Cmd[T]) (string, error) {
	if len(args) == 0 {
		return "", nil
	}
	arg := args[0]
	actcmd := ""
	for _, cmd := range cmds {
		if arg == cmd.Name {
			actcmd = arg
			ocmd, err := ParseArgs(cfg, args[1:], cmds...)
			if err != nil {
				return "", err
			}
			if ocmd != "" {
				actcmd += " " + ocmd
			}
			return actcmd, nil
		}
	}
	return "", nil
}

// ParseFlags parses the given flags using the given map of all of the
// available flags, setting the values from that map accordingly.
// Setting errNotFound = true causes flags that are not in allFlags to
// trigger an error; otherwise, it just skips those. The flags should be
// gotten through [GetArgs] first.
func ParseFlags(flags map[string]string, allFlags map[string]reflect.Value, errNotFound bool) error {
	for name, value := range flags {
		err := ParseFlag(name, value, allFlags, errNotFound)
		if err != nil {
			return err
		}
	}
	return nil
}

// ParseFlag parses the flag with the given name and the given value
// using the given map of all of the available flags, setting the value
// in that map corresponding to the flag name accordingly. Setting
// errNotFound = true causes passing a flag name that is not in allFlags
// to trigger an error; otherwise, it just does nothing and returns no error.
// It is designed for use in [ParseFlags] and should typically not be used by
// end-user code.
func ParseFlag(name string, value string, allFlags map[string]reflect.Value, errNotFound bool) error {
	fval, exists := allFlags[name]
	if !exists {
		if errNotFound {
			return fmt.Errorf("flag name not recognized: %q", name)
		}
		return nil
	}

	isBool := kit.NonPtrValue(fval).Kind() == reflect.Bool

	if isBool {
		lcnm := strings.ToLower(name)
		negate := false
		if len(lcnm) > 3 {
			if lcnm[:3] == "no_" || lcnm[:3] == "no-" {
				negate = true
			} else if lcnm[:2] == "no" {
				if _, has := allFlags[lcnm[2:]]; has { // e.g., nogui and gui is on list
					negate = true
				}
			}
		}
		if negate {
			value = "false"
		} else {
			value = "true"
		}
	}
	if value == "" {
		// got '--flag' but arg was required
		return fmt.Errorf("flag needs an argument: %q", name)
	}

	return SetArgValue(name, fval, value)
}

func ParseArgOld(s string, args []string, allArgs map[string]reflect.Value, errNotFound bool) (a []string, err error) {
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
