// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"goki.dev/grease/testdata"
	"goki.dev/grog"
	"goki.dev/grows/tomls"
)

// TestSubConfig is a sub-struct with special params
type TestSubConfig struct {

	// [def: 10] number of patterns to create
	NPats int `def:"10" desc:"number of patterns to create"`

	// [def: 0.15] proportion activity of created params
	Sparseness float32 `def:"0.15" desc:"proportion activity of created params"`
}

// TestConfig is a testing config
type TestConfig struct {

	// specify include files here, and after configuration, it contains list of include files added
	Includes []string `desc:"specify include files here, and after configuration, it contains list of include files added"`

	// [def: true] open the GUI -- does not automatically run -- if false, then runs automatically and quits
	GUI bool `def:"true" desc:"open the GUI -- does not automatically run -- if false, then runs automatically and quits"`

	// [def: true] use the GPU for computation
	GPU bool `def:"true" desc:"use the GPU for computation"`

	// log debugging information
	Debug bool `desc:"log debugging information"`

	// important for testing . notation etc
	PatParams TestSubConfig `desc:"important for testing . notation etc"`

	// network parameters applied after built-in params -- use toml map format: '{key = val, key2 = val2}' where key is 'selector:path' (e.g., '.PFCLayer:Layer.Inhib.Layer.Gi' where '.PFCLayer' is a class) and values should be strings to be consistent with standard params format
	Network map[string]any `desc:"network parameters applied after built-in params -- use toml map format: '{key = val, key2 = val2}' where key is 'selector:path' (e.g., '.PFCLayer:Layer.Inhib.Layer.Gi' where '.PFCLayer' is a class) and values should be strings to be consistent with standard params format"`

	// ParamSet name to use -- must be valid name as listed in compiled-in params or loaded params
	ParamSet string `desc:"ParamSet name to use -- must be valid name as listed in compiled-in params or loaded params"`

	// Name of the JSON file to input saved parameters from.
	ParamFile string `desc:"Name of the JSON file to input saved parameters from."`

	// Name of the file to output all parameter data. If not empty string, program should write file(s) and then exit
	ParamDocFile string `desc:"Name of the file to output all parameter data. If not empty string, program should write file(s) and then exit"`

	// extra tag to add to file names and logs saved from this run
	Tag string `desc:"extra tag to add to file names and logs saved from this run"`

	// [def: testing is fun] user note -- describe the run params etc -- like a git commit message for the run
	Note string `def:"testing is fun" desc:"user note -- describe the run params etc -- like a git commit message for the run"`

	// [def: 0] starting run number -- determines the random seed -- runs counts from there -- can do all runs in parallel by launching separate jobs with each run, runs = 1
	Run int `def:"0" desc:"starting run number -- determines the random seed -- runs counts from there -- can do all runs in parallel by launching separate jobs with each run, runs = 1"`

	// [def: 10] total number of runs to do when running Train
	Runs int `def:"10" desc:"total number of runs to do when running Train"`

	// [def: 100] total number of epochs per run
	Epochs int `def:"100" desc:"total number of epochs per run"`

	// [def: 128] total number of trials per epoch.  Should be an even multiple of NData.
	NTrials int `def:"128" desc:"total number of trials per epoch.  Should be an even multiple of NData."`

	// [def: 16] number of data-parallel items to process in parallel per trial -- works (and is significantly faster) for both CPU and GPU.  Results in an effective mini-batch of learning.
	NData int `def:"16" desc:"number of data-parallel items to process in parallel per trial -- works (and is significantly faster) for both CPU and GPU.  Results in an effective mini-batch of learning."`

	// if true, save final weights after each run
	SaveWts bool `desc:"if true, save final weights after each run"`

	// [def: true] if true, save train epoch log to file, as .epc.tsv typically
	EpochLog bool `def:"true" desc:"if true, save train epoch log to file, as .epc.tsv typically"`

	// [def: true] if true, save run log to file, as .run.tsv typically
	RunLog bool `def:"true" desc:"if true, save run log to file, as .run.tsv typically"`

	// [def: true] if true, save train trial log to file, as .trl.tsv typically. May be large.
	TrialLog bool `def:"true" desc:"if true, save train trial log to file, as .trl.tsv typically. May be large."`

	// [def: false] if true, save testing epoch log to file, as .tst_epc.tsv typically.  In general it is better to copy testing items over to the training epoch log and record there.
	TestEpochLog bool `def:"false" desc:"if true, save testing epoch log to file, as .tst_epc.tsv typically.  In general it is better to copy testing items over to the training epoch log and record there."`

	// [def: false] if true, save testing trial log to file, as .tst_trl.tsv typically. May be large.
	TestTrialLog bool `def:"false" desc:"if true, save testing trial log to file, as .tst_trl.tsv typically. May be large."`

	// if true, save network activation etc data from testing trials, for later viewing in netview
	NetData bool `desc:"if true, save network activation etc data from testing trials, for later viewing in netview"`

	// can set these values by string representation when using enumgen
	Enum testdata.TestEnum `desc:"can set these values by string representation when using enumgen"`

	// [def: [1, 2.14, 3.14]] test slice case
	Slice []float32 `def:"[1, 2.14, 3.14]" desc:"test slice case"`

	// [def: ['cat','dog one','dog two']] test string slice case
	StrSlice []string `posarg:"all" def:"['cat','dog one','dog two']" desc:"test string slice case"`
}

func (cfg *TestConfig) IncludesPtr() *[]string { return &cfg.Includes }

func TestRun(t *testing.T) {
	cfg := &TestConfig{}
	os.Args = []string{"myapp", "-gui", "install", "-no-gpu=f", "windows", "-Note", "Hello, World", "-PAT_PARAMS_SPARSENESS=4", "-net-data", "darwin"}

	// we test for correct config in and out of command through closure
	var (
		inCmd      bool
		strSlice   []string
		gpu        bool
		sparseness float32
	)
	err := Run(DefaultOptions("myapp", "My App", "My App is an awesome app"), cfg, &Cmd[*TestConfig]{
		Func: func(tc *TestConfig) error {
			inCmd = true
			strSlice = tc.StrSlice
			gpu = tc.GPU
			sparseness = tc.PatParams.Sparseness
			if tc.Note != "Hello, World" {
				t.Errorf("expeected note to be %q but got %q", "Hello, World", tc.Note)
			}
			return nil
		},
		Name: "install",
		Doc:  "install installs stuff",
	})
	if err != nil {
		t.Error(err)
	}
	if !inCmd {
		t.Errorf("never got into command")
	}
	if !reflect.DeepEqual(strSlice, []string{"windows", "darwin"}) || !gpu || sparseness != 4 {
		t.Errorf("got bad values for config (config: %#v)", cfg)
	}
}

func TestConfigFunc(t *testing.T) {
	cfg := &TestConfig{}
	os.Args = []string{"myapp", "-no-net-data", "build", "-gpu", "../main", "-Note", "Hello, World", "-v", "-PAT_PARAMS_SPARSENESS=4"}
	cmd, err := Config(DefaultOptions("myapp", "My App", "My App is an awesome app"), cfg, &Cmd[*TestConfig]{
		Func: func(tc *TestConfig) error { return nil },
		Name: "build",
		Doc:  "build builds stuff",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cmd != "build" {
		t.Errorf("expected command to be build but got %q", cmd)
	}
	if cfg.NetData || !cfg.GPU || cfg.Note != "Hello, World" || cfg.PatParams.Sparseness != 4 || !reflect.DeepEqual(cfg.StrSlice, []string{"../main"}) {
		t.Errorf("error setting configuration info (config: %#v)", cfg)
	}
	if grog.UserLevel != grog.Info {
		t.Errorf("expected grog user level to be Info but it is %v", grog.UserLevel)
	}
}

func TestDefaults(t *testing.T) {
	cfg := &TestConfig{}
	SetFromDefaults(cfg)
	if cfg.Epochs != 100 || cfg.EpochLog != true || cfg.Note != "testing is fun" {
		t.Errorf("Main defaults failed to set")
	}
	if cfg.PatParams.NPats != 10 || cfg.PatParams.Sparseness != 0.15 {
		t.Errorf("PatParams defaults failed to set")
	}
	// fmt.Printf("%#v\n", cfg.Slice)
	if len(cfg.Slice) != 3 || cfg.Slice[2] != 3.14 {
		t.Errorf("Slice defaults failed to set")
	}
	if len(cfg.StrSlice) != 3 || cfg.StrSlice[1] != "dog one" {
		t.Errorf("StrSlice defaults failed to set")
	}
}

func TestGetArgs(t *testing.T) {
	sargs := []string{"-gui", "run", "-param-set", "std", "-no-net-data", "main", "-epochs=5", "-note", "hello", "-debug=0", "-enum", "TestValue1", "-sparseness", "5.0", "-pat-params-n-pats", "28"}
	bf := BoolFlags(&TestConfig{})
	args, flags, err := GetArgs(sargs, bf)
	if err != nil {
		t.Errorf("error getting args: %v", err)
	}
	wargs := []string{"run", "main"}
	if !reflect.DeepEqual(args, wargs) {
		t.Errorf("expected args to be \n%#v\n\tbut got \n%#v", wargs, args)
	}
	wflags := map[string]string{"debug": "0", "enum": "TestValue1", "epochs": "5", "gui": "", "no-net-data": "", "note": "hello", "param-set": "std", "pat-params-n-pats": "28", "sparseness": "5.0"}
	if !reflect.DeepEqual(flags, wflags) {
		t.Errorf("expected flags to be \n%#v\n\tbut got \n%#v", wflags, flags)
	}
}

func TestArgsPrint(t *testing.T) {
	t.Skip("prints all possible args")

	cfg := &TestConfig{}
	allFields := &Fields{}
	AddFields(cfg, allFields, "")
	allFlags := &Fields{}
	AddFlags(allFields, allFlags, []string{}, map[string]string{})

	fmt.Println("Args:")
	fmt.Println(strings.Join(allFlags.Keys(), "\n"))
}

func TestSetFromArgs(t *testing.T) {
	cfg := &TestConfig{}
	err := SetFromDefaults(cfg)
	if err != nil {
		t.Error(err)
	}
	// note: cannot use "-Includes=testcfg.toml",
	args := []string{"-save-wts", "goki", "-nogui", "-no-epoch-log", "--NoRunLog", "play", "--runs=5", "--run", "1", "--TAG", "nice", "--PatParams.Sparseness=0.1", "orange", "--Network", "{'.PFCLayer:Layer.Inhib.Gi' = '2.4', '#VSPatchPrjn:Prjn.Learn.LRate' = '0.01'}", "-Enum=TestValue2", "apple", "-Slice=[3.2, 2.4, 1.9]"}

	_, err = SetFromArgs(cfg, args, ErrNotFound)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Runs != 5 || cfg.Run != 1 || cfg.Tag != "nice" || cfg.PatParams.Sparseness != 0.1 || cfg.SaveWts != true || cfg.GUI != false || cfg.EpochLog != false || cfg.RunLog != false {
		t.Errorf("args not set properly (config: %#v)", cfg)
	}
	if cfg.Enum != testdata.TestValue2 {
		t.Errorf("expected args enum from string to be \n%v\n\tbut got \n%v", testdata.TestValue2, cfg.Enum)
	}
	wcs := []float32{3.2, 2.4, 1.9}
	if !reflect.DeepEqual(cfg.Slice, wcs) {
		t.Errorf("expected args slice to be \n%#v\n\tbut got \n%#v", wcs, cfg.Slice)
	}
	wcss := []string{"goki", "play", "orange", "apple"}
	if !reflect.DeepEqual(cfg.StrSlice, wcss) {
		t.Errorf("expected args string slice to be \n%#v\n\tbut got \n%#v", wcss, cfg.StrSlice)
	}
	// if cfg.Network != nil {
	// 	mv := cfg.Network
	// 	for k, v := range mv {
	// 		fmt.Println(k, " = ", v)
	// 	}
	// }
}

// TestSetFromArgsErr ensures we get errors for various problems with arguments
func TestSetFromArgsErr(t *testing.T) {
	cfg := &TestConfig{}
	err := SetFromDefaults(cfg)
	if err != nil {
		t.Error(err)
	}

	args := [][]string{
		{"-save-wts", "goki", "---runs=5", "apple"},
		{"--No RunLog", "-note", "hi"},
		{"-sparsenes", "0.1"},
		{"-runs={"},
		{"--net-data=me"},
	}

	for i, a := range args {
		_, err = SetFromArgs(cfg, a, ErrNotFound)
		if err == nil {
			t.Errorf("expected error but got none for args %v (index %d)", a, i)
		}
	}
}

// TestUnusedArg ensures we error correctly on an unused argument
func TestUnusedArg(t *testing.T) {
	type myType struct {
		Name    string
		Age     int
		LikesGo bool
	}

	cfg := &myType{}
	args := []string{"-name", "Go Gopher", "-likes-go", "main", "-age=13"}
	_, err := SetFromArgs(cfg, args, ErrNotFound)
	if err == nil || !strings.Contains(err.Error(), "unused arguments") { // hacky logic but fine for simple test
		t.Errorf("expected to get unused arguments error, but got err = %v", err)
	}
}

// TestNoErrNotFound ensures we get no unrecognized flag name errors with NoErrNotFound
func TestNoErrNotFound(t *testing.T) {
	cfg := &TestConfig{}
	args := []string{"-sparsenes", "0.1"}
	_, err := SetFromArgs(cfg, args, NoErrNotFound)
	if err != nil {
		t.Errorf("expected to get no error with NoErrNotFound, but got err = %v", err)
	}
}

func TestOpen(t *testing.T) {
	opts := DefaultOptions("test", "Test", "")
	opts.IncludePaths = []string{".", "testdata"}
	cfg := &TestConfig{}
	err := OpenWithIncludes(opts, cfg, "testcfg.toml")
	if err != nil {
		t.Errorf(err.Error())
	}

	// fmt.Println("includes:", cfg.Includes)

	// if cfg.Network != nil {
	// 	mv := cfg.Network
	// 	for k, v := range mv {
	// 		fmt.Println(k, " = ", v)
	// 	}
	// }

	if cfg.GUI != true || cfg.Tag != "testing" {
		t.Errorf("testinc.toml not parsed\n")
	}
	if cfg.Epochs != 500 || cfg.GPU != true {
		t.Errorf("testinc2.toml not parsed\n")
	}
	if cfg.Note != "something else" {
		t.Errorf("testinc3.toml not parsed\n")
	}
	if cfg.Runs != 8 {
		t.Errorf("testinc3.toml didn't overwrite testinc2\n")
	}
	if cfg.NTrials != 32 {
		t.Errorf("testinc.toml didn't overwrite testinc2\n")
	}
	if cfg.NData != 12 {
		t.Errorf("testcfg.toml didn't overwrite testinc3\n")
	}
	if cfg.Enum != testdata.TestValue2 {
		t.Errorf("testinc.toml Enum value not parsed\n")
	}
}

func TestUsage(t *testing.T) {
	t.Skip("prints usage string")
	cfg := &TestConfig{}
	us := Usage(DefaultOptions("test", "Test", ""), cfg, "")
	fmt.Println(us)
}

func TestSave(t *testing.T) {
	// t.Skip("prints usage string")
	opts := DefaultOptions("test", "Test", "")
	opts.IncludePaths = []string{".", "testdata"}
	cfg := &TestConfig{}
	OpenWithIncludes(opts, cfg, "testcfg.toml")
	tomls.Save(cfg, "testdata/testwrite.toml")
}

func TestConfigOpen(t *testing.T) {
	// t.Skip("prints usage string")
	opts := DefaultOptions("test", "Test", "")
	opts.IncludePaths = []string{".", "testdata"}
	opts.NeedConfigFile = true
	cfg := &TestConfig{}
	_, err := Config(opts, cfg)
	if err == nil {
		t.Errorf("should have Config error")
		// } else {
		// 	fmt.Println(err)
	}
	opts.DefaultFiles = []string{"aldfkj.toml"}
	_, err = Config(opts, cfg)
	if err == nil {
		t.Errorf("should have Config error")
		// } else {
		// 	fmt.Println(err)
	}
	opts.DefaultFiles = []string{"aldfkj.toml", "testcfg.toml"}
	_, err = Config(opts, cfg)
	if err != nil {
		t.Error(err)
	}
}
