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
	"strconv"
	"strings"

	"github.com/iancoleman/strcase"
	"goki.dev/ki/v2/kit"
	"goki.dev/ki/v2/toml"
	"golang.org/x/exp/slices"
)

// SetFromArgs sets Config values from command-line args,
// based on the field names in the Config struct.
// Returns any args that did not start with a `-` flag indicator.
// For more robust error processing, it is assumed that all flagged args (-)
// must refer to fields in the config, so any that fail to match trigger
// an error. Errors can also result from parsing.
func SetFromArgs[T any](cfg T, args []string, cmds ...*Cmd[T]) (string, error) {
	nfargs, flags, err := GetArgs(args)
	if err != nil {
		return "", err
	}
	cmd, allFlags, err := ParseArgs(cfg, nfargs, cmds...)
	if err != nil {
		return "", err
	}
	err = ParseFlags(flags, allFlags, true)
	if err != nil {
		return "", err
	}
	return cmd, nil
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
// gotten through [GetArgs] first. It returns the (sub)command specified by
// the arguments, a map from all of the flag names to their associated
// settable values, and any error.
func ParseArgs[T any](cfg T, args []string, cmds ...*Cmd[T]) (cmd string, allFlags map[string]reflect.Value, err error) {
	allFlags = map[string]reflect.Value{}
	CommandFlags(allFlags)
	newArgs, newCmd, err := parseArgsImpl(cfg, args, allFlags, "", cmds...)
	if err != nil {
		return newCmd, allFlags, err
	}

	newArgs, err = FieldFlagNames(cfg, allFlags, newCmd, newArgs)
	if err != nil {
		return newCmd, allFlags, fmt.Errorf("error getting field flag names: %w", err)
	}
	if len(newArgs) > 0 {
		return newCmd, allFlags, fmt.Errorf("got unused arguments: %v", newArgs)
	}
	return newCmd, allFlags, nil
}

// parseArgsImpl is the underlying implementation of [ParseArgs] that is called
// recursively and takes everything [ParseArgs] does and the current flags and
// command state, and returns everything [ParseArgs] does and the args state.
func parseArgsImpl[T any](cfg T, baseArgs []string, allFlags map[string]reflect.Value, baseCmd string, cmds ...*Cmd[T]) (args []string, cmd string, err error) {
	// we start with our base args and command
	args = baseArgs
	cmd = baseCmd

	// if we have no additional args, we have nothing to do
	if len(args) == 0 {
		return
	}

	// we only care about one arg at a time (everything else is handled recursively)
	arg := args[0]
	// get all of the (sub)commands in our base command
	baseCmdStrs := strings.Fields(baseCmd)
	for _, c := range cmds {
		// get all of the (sub)commands in this command
		cmdStrs := strings.Fields(c.Name)
		// find the (sub)commands that our base command shares with the command we are checking
		gotTo := 0
		hasMismatch := false
		for i, cstr := range cmdStrs {
			// if we have no more (sub)commands on our base, mark our location and break
			if i >= len(baseCmdStrs) {
				gotTo = i
				break
			}
			// if we have a different thing than our base, it is a mismatch
			if baseCmdStrs[i] != cstr {
				hasMismatch = true
				break
			}
		}
		// if we have a different sub(command) for something, this isn't the right command
		if hasMismatch {
			continue
		}
		// if the thing after we ran out of (sub)commands on our base isn't our next arg, this isn't the right command
		if arg != cmdStrs[gotTo] {
			continue
		}
		// otherwise, it is the right command, and our new command is our base plus our next arg
		cmd = arg
		if baseCmd != "" {
			cmd = baseCmd + " " + arg
		}
		// we have consumed our next arg, so we get rid of it
		args = args[1:]
		// then, we recursively parse again with our new command as context
		oargs, ocmd, err := parseArgsImpl(cfg, args, allFlags, cmd, cmds...)
		if err != nil {
			return nil, "", err
		}
		// our new args and command are now whatever the recursive call returned, building upon what we passed it
		args = oargs
		cmd = ocmd
		// we got the command we wanted, so we can break
		break
	}
	return
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
			return err
		}
		err = kit.CopyMapRobust(fval.Interface(), mval["tmp"])
		if err != nil {
			return fmt.Errorf("not able to set map field from arg: %q val: %q: %w", name, value, err)
		}
	case vk == reflect.Slice:
		mval := make(map[string]any)
		err := toml.ReadBytes(&mval, []byte("tmp="+value)) // use toml decoder
		if err != nil {
			return err
		}
		err = kit.CopySliceRobust(fval.Interface(), mval["tmp"])
		if err != nil {
			return fmt.Errorf("not able to set slice field from arg: %q val: %q: %w", name, value, err)
		}
	default:
		ok := kit.SetRobust(fval.Interface(), value) // overkill but whatever
		if !ok {
			return fmt.Errorf("not able to set field from arg: %q val: %q", name, value)
		}
	}
	return nil
}

// FieldFlagNames adds to given flags map all the different ways the field names
// can be specified as arg flags, mapping to the reflect.Value. It also uses
// the given positional arguments to set the values of the object based on any
// posarg struct tags that fields have. The posarg struct tag must be either
// "all" or a valid uint.
func FieldFlagNames(obj any, allFlags map[string]reflect.Value, cmd string, args []string) ([]string, error) {
	return fieldFlagNamesStruct(obj, "", false, allFlags, cmd, args)
}

func addAllCases(nm, path string, pval reflect.Value, allFlags map[string]reflect.Value, cmd string) {
	if nm == "Includes" {
		return // skip
	}
	if path != "" {
		nm = path + "." + nm
	}
	allFlags[nm] = pval
	allFlags[strings.ToLower(nm)] = pval
	allFlags[strcase.ToKebab(nm)] = pval
	allFlags[strcase.ToSnake(nm)] = pval
	allFlags[strcase.ToScreamingSnake(nm)] = pval
}

// fieldFlagNamesStruct returns map of all the different ways the field names
// can be specified as arg flags, mapping to the reflect.Value
func fieldFlagNamesStruct(obj any, path string, nest bool, allFlags map[string]reflect.Value, cmd string, args []string) ([]string, error) {
	if kit.IfaceIsNil(obj) {
		return nil, nil
	}
	ov := reflect.ValueOf(obj)
	if ov.Kind() == reflect.Pointer && ov.IsNil() {
		return nil, nil
	}
	leftovers := args
	val := kit.NonPtrValue(ov)
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		fv := val.Field(i)
		pval := kit.PtrValue(fv)
		cmdtag, ok := f.Tag.Lookup("cmd")
		if ok && cmdtag != cmd { // if we are associated with a different command, skip
			continue
		}
		posargtag, ok := f.Tag.Lookup("posarg")
		if ok {
			if posargtag == "all" {
				ok := kit.SetRobust(pval.Interface(), args)
				if !ok {
					return nil, fmt.Errorf("not able to set field %q to all positional arguments: %v", f.Name, args)
				}
				leftovers = []string{} // everybody has been consumed
			} else {
				ui, err := strconv.ParseUint(posargtag, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("invalid value %q for posarg struct tag on field %q: %w", posargtag, f.Name, err)
				}
				if ui >= uint64(len(args)) {
					return nil, fmt.Errorf("missing positional argument %d used for field %q", ui, f.Name)
				}
				err = SetArgValue(f.Name, pval, args[ui]) // must be pointer to be settable
				if err != nil {
					return nil, fmt.Errorf("error setting field %q to positional argument %d (%q): %w", f.Name, ui, args[ui], err)
				}
				leftovers = slices.Delete(leftovers, int(ui), int(ui+1)) // we have consumed this argument
			}

		}
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
			fieldFlagNamesStruct(kit.PtrValue(fv).Interface(), nwPath, nwNest, allFlags, cmd, args)
			continue
		}
		addAllCases(f.Name, path, pval, allFlags, cmd)
		if f.Type.Kind() == reflect.Bool {
			addAllCases("No"+f.Name, path, pval, allFlags, cmd)
		}
		// now process adding non-nested version of field
		if path == "" || nest {
			continue
		}
		neststr, ok := f.Tag.Lookup("nest")
		if ok && (neststr == "+" || neststr == "true") {
			continue
		}
		if _, has := allFlags[f.Name]; has {
			fmt.Printf("warning: programmer error: grease config field \"%s.%s\" cannot be added as a non-nested flag with the name %q because that name has already been registered by another field; add the field tag 'nest:\"+\"' to the field you want to require nested access for (ie: \"Path.Field\" instead of \"Field\") to remove this warning\n", path, f.Name, f.Name)
			continue
		}
		addAllCases(f.Name, "", pval, allFlags, cmd)
		if f.Type.Kind() == reflect.Bool {
			addAllCases("No"+f.Name, "", pval, allFlags, cmd)
		}
	}
	return leftovers, nil
}

// CommandFlags adds non-field flags that control the config process
// to the given map of flags. These flags have no actual effect and
// map to a placeholder value because they are handled elsewhere, but
// they must be set to prevent errors. The following flags are added:
//
//	-config -cfg -help -h
func CommandFlags(allFlags map[string]reflect.Value) {
	val := ""
	allFlags["config"] = reflect.ValueOf(&val)
	allFlags["cfg"] = reflect.ValueOf(&val)
	allFlags["help"] = reflect.ValueOf(&val)
	allFlags["h"] = reflect.ValueOf(&val)
}
