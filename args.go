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
	"goki.dev/ki/v2/toml"
	"goki.dev/laser"
)

var (
	// ErrNotFound can be passed to [SetFromArgs] and [ParseFlags]
	// to indicate that they should return an error for a flag that
	// is set but not found in the configuration struct.
	ErrNotFound = true
	// ErrNotFound can be passed to [SetFromArgs] and [ParseFlags]
	// to indicate that they should NOT return an error for a flag that
	// is set but not found in the configuration struct.
	NoErrNotFound = false
)

// SetFromArgs sets Config values from command-line args,
// based on the field names in the Config struct.
// Returns any args that did not start with a `-` flag indicator.
// If errNotFound is set to true, it is assumed that all flagged args (-)
// must refer to fields in the config, so any that fail to match trigger
// an error. Errors can also result from parsing.
func SetFromArgs[T any](cfg T, args []string, errNotFound bool, cmds ...*Cmd[T]) (string, error) {
	nfargs, flags, err := GetArgs(args, BoolFlags(cfg))
	if err != nil {
		return "", err
	}
	cmd, allFlags, err := ParseArgs(cfg, nfargs, flags, cmds...)
	if err != nil {
		return "", err
	}
	err = ParseFlags(flags, allFlags, errNotFound)
	if err != nil {
		return "", err
	}
	return cmd, nil
}

// BoolFlags returns a map with a true value for every flag name
// that maps to a boolean field. This is needed so that bool
// flags can be properly set with their shorthand syntax.
// It should only be needed for internal use and not end-user code.
func BoolFlags(obj any) map[string]bool {
	fields := &Fields{}
	AddFields(obj, fields, AddAllFields)

	res := map[string]bool{}

	for _, kv := range fields.Order {
		f := kv.Val

		if f.Field.Type.Kind() != reflect.Bool { // we only care about bools here
			continue
		}

		// we need all cases of both normal and "no" version for all names
		for _, name := range f.Names {
			res[name] = true
			res[strings.ToLower(name)] = true
			res[strcase.ToKebab(name)] = true
			res[strcase.ToSnake(name)] = true
			res[strcase.ToScreamingSnake(name)] = true

			nnm := "No" + name
			res[nnm] = true
			res[strings.ToLower(nnm)] = true
			res[strcase.ToKebab(nnm)] = true
			res[strcase.ToSnake(nnm)] = true
			res[strcase.ToScreamingSnake(nnm)] = true
		}
	}
	return res
}

// GetArgs processes the given args using map of all available args,
// returning the leftover (positional) args, the flags, and any error.
// setting errNotFound = true causes args that are not in allArgs to
// trigger an error.  Otherwise, it just skips those.
func GetArgs(args []string, boolFlags map[string]bool) ([]string, map[string]string, error) {
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
		name, value, nargs, err := GetFlag(s, args, boolFlags)
		if err != nil {
			return nonFlags, flags, err
		}
		// we need to updated remaining args with latest
		args = nargs
		if name != "" { // we ignore no-names so that we can skip things like test args
			flags[name] = value
		}
	}
	return nonFlags, flags, nil
}

// GetFlag parses the given flag arg string in the context
// of the given remaining arguments, and returns the
// name of the flag, the value of the flag, the remaining
// arguments updated with any changes caused by getting
// this flag, and any error. It is designed for use in [GetArgs]
// and should typically not be used by end-user code.
func GetFlag(s string, args []string, boolFlags map[string]bool) (name, value string, a []string, err error) {
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
		name = ""
		return
	}

	// split on equal (we could be in the form flag=value)
	split := strings.SplitN(name, "=", 2)
	name = split[0]
	if len(split) == 2 {
		// if we are in the form flag=value, we are done
		value = split[1]
	} else if len(a) > 0 && !boolFlags[name] { // otherwise, if we still have more remaining args and are not a bool, our value could be the next arg (if we are a bool, we don't care about the next value)
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
func ParseArgs[T any](cfg T, args []string, flags map[string]string, cmds ...*Cmd[T]) (cmd string, allFlags *Fields, err error) {
	newArgs, newCmd, err := parseArgsImpl(cfg, args, "", cmds...)
	if err != nil {
		return newCmd, allFlags, err
	}

	allFields := &Fields{}
	CommandFields(allFields)
	AddFields(cfg, allFields, newCmd)

	allFlags = &Fields{}
	newArgs, err = AddFlags(allFields, allFlags, newCmd, newArgs, flags)
	if err != nil {
		return newCmd, allFields, err
	}
	if len(newArgs) > 0 {
		return newCmd, allFields, fmt.Errorf("got unused arguments: %v", newArgs)
	}
	return newCmd, allFlags, nil
}

// parseArgsImpl is the underlying implementation of [ParseArgs] that is called
// recursively and takes everything [ParseArgs] does and the current flags and
// command state, and returns everything [ParseArgs] does and the args state.
func parseArgsImpl[T any](cfg T, baseArgs []string, baseCmd string, cmds ...*Cmd[T]) (args []string, cmd string, err error) {
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
		oargs, ocmd, err := parseArgsImpl(cfg, args, cmd, cmds...)
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
func ParseFlags(flags map[string]string, allFlags *Fields, errNotFound bool) error {
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
func ParseFlag(name string, value string, allFlags *Fields, errNotFound bool) error {
	f, exists := allFlags.ValByKeyTry(name)
	if !exists {
		if errNotFound {
			return fmt.Errorf("flag name %q not recognized", name)
		}
		return nil
	}

	isBool := laser.NonPtrValue(f.Value).Kind() == reflect.Bool

	if isBool {
		// check if we have a "no" prefix and set negate based on that
		lcnm := strings.ToLower(name)
		negate := false
		if len(lcnm) > 3 {
			if lcnm[:3] == "no_" || lcnm[:3] == "no-" {
				negate = true
			} else if lcnm[:2] == "no" {
				if _, has := allFlags.ValByKeyTry(lcnm[2:]); has { // e.g., nogui and gui is on list
					negate = true
				}
			}
		}
		// the value could be explicitly set to a bool value,
		// so we check that; if it is not set, it is true
		b := true
		if value != "" {
			var err error
			b, err = strconv.ParseBool(value)
			if err != nil {
				return fmt.Errorf("error parsing bool flag %q: %w", name, err)
			}
		}
		// if we are negating and true (ex: -no-something), or not negating
		// and false (ex: -something=false), we are false
		if negate && b || !negate && !b {
			value = "false"
		} else { // otherwise, we are true
			value = "true"
		}
	}
	if value == "" {
		// got '--flag' but arg was required
		return fmt.Errorf("flag %q needs an argument", name)
	}

	return SetFieldValue(f, value)
}

// SetFieldValue sets the value of the given field to the given value
func SetFieldValue(f *Field, value string) error {
	nptyp := laser.NonPtrType(f.Value.Type())
	vk := nptyp.Kind()
	switch {
	case vk == reflect.Map:
		mval := make(map[string]any)
		err := toml.ReadBytes(&mval, []byte("tmp="+value)) // use toml decoder
		if err != nil {
			return err
		}
		err = laser.CopyMapRobust(f.Value.Interface(), mval["tmp"])
		if err != nil {
			return fmt.Errorf("not able to set map field %q from flag value %q: %w", f.Name, value, err)
		}
	case vk == reflect.Slice:
		mval := make(map[string]any)
		err := toml.ReadBytes(&mval, []byte("tmp="+value)) // use toml decoder
		if err != nil {
			return err
		}
		err = laser.CopySliceRobust(f.Value.Interface(), mval["tmp"])
		if err != nil {
			return fmt.Errorf("not able to set slice field %q from flag value %q: %w", f.Name, value, err)
		}
	default:
		ok := laser.SetRobust(f.Value.Interface(), value) // overkill but whatever
		if !ok {
			return fmt.Errorf("not able to set field %q from flag value %q", f.Name, value)
		}
	}
	return nil
}

// addAllCases adds all string cases of the given field with the
// given name to the given set of flags.
func addAllCases(nm string, field *Field, allFlags *Fields) {
	if nm == "Includes" {
		return // skip Includes
	}
	allFlags.Add(nm, field)
	allFlags.Add(strings.ToLower(nm), field)
	allFlags.Add(strcase.ToKebab(nm), field)
	allFlags.Add(strcase.ToSnake(nm), field)
	allFlags.Add(strcase.ToScreamingSnake(nm), field)
}

// allCases returns all of the string cases of the given name
func allCases(nm string) []string {
	return []string{
		nm, strings.ToLower(nm), strcase.ToKebab(nm), strcase.ToSnake(nm), strcase.ToScreamingSnake(nm),
	}
}

// AddFlags adds to given flags map all the different ways the field names
// can be specified as arg flags, mapping to the reflect.Value. It also uses
// the given positional arguments to set the values of the object based on any
// posarg struct tags that fields have. The posarg struct tag must be either
// "all" or a valid uint.
func AddFlags(allFields *Fields, allFlags *Fields, cmd string, args []string, flags map[string]string) ([]string, error) {
	consumed := map[int]bool{} // which args we have consumed via pos args
	for _, kv := range allFields.Order {
		v := kv.Val
		f := v.Field

		for _, name := range v.Names {
			addAllCases(name, v, allFlags)
			if f.Type.Kind() == reflect.Bool {
				addAllCases("No"+name, v, allFlags)
			}
		}

		// set based on pos arg
		posArgTag, ok := f.Tag.Lookup("posarg")
		if ok {
			if posArgTag == "all" {
				ok := laser.SetRobust(v.Value.Interface(), args)
				if !ok {
					return nil, fmt.Errorf("not able to set field %q to all positional arguments: %v", f.Name, args)
				}
				// everybody has been consumed
				for i := range args {
					consumed[i] = true
				}
			} else {
				ui, err := strconv.ParseUint(posArgTag, 10, 64)
				if err != nil {
					return nil, fmt.Errorf("programmer error: invalid value %q for posarg struct tag on field %q: %w", posArgTag, f.Name, err)
				}
				// if this is true, the pos arg is missing
				if ui >= uint64(len(args)) {
					// check if we have set this pos arg as a flag; if we have,
					// it makes up for the missing pos arg and there is no error,
					// but otherwise there is an error
					got := false
					for _, fnm := range v.Names { // TODO: is there a more efficient way to do this?
						for _, cnm := range allCases(fnm) {
							_, ok := flags[cnm]
							if ok {
								got = true
								break
							}
						}
						if got {
							break
						}
					}
					if got {
						continue // if we got the pos arg through the flag, we skip the rest of the pos arg stuff and go onto the next field
					} else {
						return nil, fmt.Errorf("missing positional argument %d (%s)", ui, strcase.ToKebab(v.Names[0]))
					}
				}
				err = SetFieldValue(v, args[ui]) // must be pointer to be settable
				if err != nil {
					return nil, fmt.Errorf("error setting field %q to positional argument %d (%q): %w", f.Name, ui, args[ui], err)
				}
				consumed[int(ui)] = true // we have consumed this argument
			}
		}
	}
	// get leftovers based on who was consumed
	leftovers := []string{}
	for i, a := range args {
		if !consumed[i] {
			leftovers = append(leftovers, a)
		}
	}
	return leftovers, nil
}

// CommandFields adds non-field flags that control the config process
// to the given map of flags. These flags have no actual effect and
// map to a placeholder value because they are handled elsewhere, but
// they must be set to prevent errors. The following flags are added:
//
//	-config -cfg -help -h
func CommandFields(allFields *Fields) {
	// NOTE: we could do this through AddFields, but that
	// causes problems with the HelpCmd field capturing
	// everything, so it easier to just add manually.
	// TODO: maybe improve the structure of this

	mc := reflect.TypeOf(&MetaConfig{}).Elem()

	hf, ok := mc.FieldByName("Help")
	if !ok {
		panic("programmer error: Help field not found in MetaConfig")
	}
	hv := false
	allFields.Add("MetaConfig.Help", &Field{
		Field: hf,
		Value: reflect.ValueOf(&hv),
		Name:  "MetaConfig.Help",
		Names: []string{"help", "h"},
	})

	cf, ok := mc.FieldByName("Config")
	if !ok {
		panic("programmer error: Config field not found in MetaConfig")
	}
	cv := ""
	allFields.Add("MetaConfig.Config", &Field{
		Field: cf,
		Value: reflect.ValueOf(&cv),
		Name:  "MetaConfig.Config",
		Names: []string{"config", "cfg"},
	})
}
