// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package grease

import (
	"fmt"
	"strings"
	"testing"
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

	// can set these values by string representation if stringer and registered as an enum with kit
	Enum TestEnum `desc:"can set these values by string representation if stringer and registered as an enum with kit"`

	// [def: [1, 2.14, 3.14]] test slice case
	Slice []float32 `def:"[1, 2.14, 3.14]" desc:"test slice case"`

	// [def: ['cat','dog one','dog two']] test string slice case
	StrSlice []string `def:"['cat','dog one','dog two']" desc:"test string slice case"`
}

func (cfg *TestConfig) IncludesPtr() *[]string { return &cfg.Includes }

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
	sargs := []string{"build", "main", "-o", "-dir", "../grease", "-v", "-platform", "windows/amd64"}
	args, flags, err := GetArgs(sargs, map[string]bool{})
	if err != nil {
		t.Errorf("error getting args: %v", err)
	}
	fmt.Println(args, "\n", flags)
}

func TestArgsPrint(t *testing.T) {
	t.Skip("prints all possible args")

	cfg := &TestConfig{}
	allFields := &Fields{}
	AddFields(cfg, allFields, "")
	allFlags := &Fields{}
	AddFlags(allFields, allFlags, "", []string{}, map[string]string{})

	fmt.Println("Args:")
	fmt.Println(strings.Join(allFlags.Keys(), "\n"))
}

func TestArgs(t *testing.T) {
	cfg := &TestConfig{}
	SetFromDefaults(cfg)
	// note: cannot use "-Includes=testcfg.toml",
	args := []string{"-save-wts", "-nogui", "-no-epoch-log", "--NoRunLog", "--runs=5", "--run", "1", "--TAG", "nice", "--PatParams.Sparseness=0.1", "--Network", "{'.PFCLayer:Layer.Inhib.Gi' = '2.4', '#VSPatchPrjn:Prjn.Learn.LRate' = '0.01'}", "-Enum=TestValue2", "-Slice=[3.2, 2.4, 1.9]"}

	_, err := SetFromArgs(cfg, args, ErrNotFound)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Runs != 5 || cfg.Run != 1 || cfg.Tag != "nice" || cfg.PatParams.Sparseness != 0.1 || cfg.SaveWts != true || cfg.GUI != false || cfg.EpochLog != false || cfg.RunLog != false {
		t.Errorf("args not set properly: %#v", cfg)
	}
	if cfg.Enum != TestValue2 {
		t.Errorf("args enum from string not set properly: %#v", cfg)
	}
	if len(cfg.Slice) != 3 || cfg.Slice[2] != 1.9 {
		t.Errorf("args Slice not set properly: %#v", cfg)
	}

	// if cfg.Network != nil {
	// 	mv := cfg.Network
	// 	for k, v := range mv {
	// 		fmt.Println(k, " = ", v)
	// 	}
	// }
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
	if cfg.Enum != TestValue2 {
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
	Save(cfg, "testdata/testwrite.toml")
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
