// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

//go:generate core generate

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/reflectx"
	"github.com/cogentcore/webgpu/wgpu"
)

func init() {
	// Don't remove this; it is needed to silence various irrelevant wgpu warnings
	SetDebug(false)
}

var (
	// Debug is whether to enable debug mode, getting
	// more diagnostic output about GPU configuration and rendering.
	// It should be set using [SetDebug].
	Debug = false

	// DebugAdapter provides detailed information about the selected
	// GPU adpater device (i.e., the type and limits of the hardware).
	DebugAdapter = false

	// SelectAdapter selects the given adapter number if >= 0.
	// If there are multiple discrete gpu adapters, then all of them
	// are listed in order 0..n-1.
	SelectAdapter = -1

	// theInstance is the initialized WebGPU instance, initialized
	// for the first call to NewGPU.
	theInstance *wgpu.Instance
)

// Instance returns the highest-level GPU handle: the Instance.
func Instance() *wgpu.Instance {
	if theInstance == nil {
		theInstance = wgpu.CreateInstance(nil)
	}
	return theInstance
}

// ReleaseInstance should only be called at final termination.
func ReleaseInstance() {
	if theInstance != nil {
		theInstance.Release()
		theInstance = nil
	}
}

// SetDebug sets [Debug] (debug mode). If it is set to true,
// it calls [wgpu.SetLogLevel]([wgpu.LogLevelDebug]). Otherwise,
// it calls [wgpu.SetLogLevel]([wgpu.LogLevelError]).
// It is called automatically with false in init().
// You can also manually set the log level with
// [wgpu.SetLogLevel].
func SetDebug(debug bool) {
	Debug = debug
	if Debug {
		wgpu.SetLogLevel(wgpu.LogLevelDebug)
	} else {
		wgpu.SetLogLevel(wgpu.LogLevelError)
	}
}

// GPU represents the GPU hardware
type GPU struct {
	// GPU represents the specific GPU hardware device used.
	// You can call GetInfo() to get info.
	GPU *wgpu.Adapter

	// name of the physical GPU device
	DeviceName string

	// Properties are the general properties of the GPU adapter.
	Properties wgpu.AdapterInfo

	// Limits are the limits of the current GPU adapter.
	Limits wgpu.SupportedLimits

	// ComputeOnly indicates if this GPU is only used for compute,
	// which determines if it listens to GPU_COMPUTE_DEVICE
	// environment variable, allowing different compute devices to be
	// selected vs. graphics devices.
	ComputeOnly bool

	// maximum number of compute threads per compute shader invocation,
	// for a 1D number of threads per Warp, which is generally greater
	// than MaxComputeWorkGroup, which allows for the maxima as well.
	// This is not defined anywhere in the formal spec, unfortunately,
	// but has been determined empirically for Mac and NVIDIA which are
	// two of the most relevant use-cases.  If not a known case,
	// the MaxComputeWorkGroupvalue is used, which can significantly
	// slow down compute processing if more could actually be used.
	// Please file an issue or PR for other GPUs with known larger values.
	MaxComputeWorkGroupCount1D int
}

// NewGPU returns a new GPU, configured and ready to use.
// If only doing compute, use [NewComputeGPU].
// The surface can be used to select an appropriate adapter, and
// is recommended but not essential. Returns nil and logs an error
// if the current platform is not supported by WebGPU.
func NewGPU(sf *wgpu.Surface) *GPU {
	gp := &GPU{}
	if errors.Log(gp.init(sf)) == nil {
		return gp
	}
	return nil
}

// NewComputeGPU returns a new GPU, configured and ready to use,
// for purely compute use, which causes it to listen to
// use the GPU_COMPUTE_DEVICE variable for which GPU device to use.
// Returns nil and logs an error if the current platform
// is not supported by WebGPU.
func NewComputeGPU() *GPU {
	gp := &GPU{}
	gp.ComputeOnly = true
	if errors.Log(gp.init(nil)) == nil {
		return gp
	}
	return nil
}

// init configures the GPU
func (gp *GPU) init(sf *wgpu.Surface) error {
	inst := Instance()
	if inst == nil {
		return errors.New("WebGPU is not supported on this machine: could not create an instance")
	}
	gpIndex := 0
	if gp.ComputeOnly {
		gpus := inst.EnumerateAdapters(nil)
		if len(gpus) == 0 {
			return errors.New("WebGPU is not supported on this machine: no adapters available")
		}
		gpIndex = gp.SelectComputeGPU(gpus)
		gp.GPU = gpus[gpIndex]
	} else {
		gpus := inst.EnumerateAdapters(nil)
		if len(gpus) == 0 {
			return errors.New("WebGPU is not supported on this machine: no adapters available")
		}
		gpIndex = gp.SelectGraphicsGPU(gpus)
		gp.GPU = gpus[gpIndex]
		// note: below is a more standard way of doing it, but until we fix the issues
		// with NVIDIA adapters on linux (#1247), we are using our custom logic.
		//
		// opts := &wgpu.RequestAdapterOptions{
		// 	CompatibleSurface: sf,
		// 	PowerPreference:   wgpu.PowerPreferenceHighPerformance,
		// }
		// ad, err := inst.RequestAdapter(opts)
		// if errors.Log(err) != nil {
		// 	return err
		// }
		// gp.GPU = ad
	}
	gp.Properties = gp.GPU.GetInfo()
	gp.DeviceName = adapterName(&gp.Properties)
	if Debug || DebugAdapter {
		fmt.Println("gpu: Selected Device:", gpIndex, gp.DeviceName, " (set DebugAdapter to get more adapter info)")
	}

	gp.Limits = gp.GPU.GetLimits()

	if DebugAdapter {
		fmt.Println(gp.PropertiesString())
	}

	gp.MaxComputeWorkGroupCount1D = int(gp.Limits.Limits.MaxComputeWorkgroupsPerDimension)
	dv := actualVendorName(&gp.Properties)
	if Debug || DebugAdapter {
		fmt.Println("GPU device vendor:", dv)
	}
	if dv == "nvidia" {
		// all NVIDIA are either 1 << 31 or -1 of that.
		gp.MaxComputeWorkGroupCount1D = (1 << 31) - 1
	} else if dv == "apple" {
		gp.MaxComputeWorkGroupCount1D = (1 << 31) - 1
	}
	// note: if known to be higher for any specific case, please file an issue or PR
	// todo: where are the errors!?
	return nil
}

// actualVendorName returns the actual vendor name from the coded
// string that the adapter VendorName contains,
// or, failing that, from the description.
func actualVendorName(ai *wgpu.AdapterInfo) string {
	nm := strings.ToLower(ai.VendorName)
	// source: https://www.reddit.com/r/vulkan/comments/4ta9nj/is_there_a_comprehensive_list_of_the_names_and/
	switch nm {
	case "0x10de":
		return "nvidia"
	case "0x1002":
		return "amd"
	case "0x1010":
		return "imgtec"
	case "0x13b5":
		return "arm"
	case "0x5143":
		return "qualcomm"
	case "0x8086":
		return "intel"
	}
	vd := strings.ToLower(ai.DriverDescription)
	if strings.Contains(vd, "apple") {
		return "apple"
	}
	return nm
}

func adapterName(ai *wgpu.AdapterInfo) string {
	if ai.Name != "" && !strings.HasPrefix(ai.Name, "0x") {
		return ai.Name
	}
	if ai.DriverDescription != "" && !strings.HasPrefix(ai.DriverDescription, "0x") {
		return ai.DriverDescription
	}
	return ai.VendorName
}

func (gp *GPU) SelectGraphicsGPU(gpus []*wgpu.Adapter) int {
	n := len(gpus)
	if n == 1 {
		return 0
	}

	if SelectAdapter >= 0 && SelectAdapter < n {
		return SelectAdapter
	}
	trgDevNm := ""
	if ev := os.Getenv("GPU_DEVICE"); ev != "" {
		trgDevNm = ev
	}
	if trgDevNm != "" {
		idx, err := strconv.Atoi(trgDevNm)
		if err == nil && idx >= 0 && idx < n {
			return idx
		}
		for gi := range n {
			props := gpus[gi].GetInfo()
			if gpuIsBadBackend(props.BackendType) {
				continue
			}
			pnm := adapterName(&props)
			if strings.Contains(pnm, trgDevNm) {
				devNm := props.Name
				if Debug {
					log.Printf("gpu: selected device named: %s, specified in GPU_DEVICE or GPU_COMPUTE_DEVICE environment variable, index: %d\n", devNm, gi)
				}
				return gi
			}
		}
		if Debug {
			log.Printf("gpu: unable to find device named: %s, specified in GPU_DEVICE or GPU_COMPUTE_DEVICE environment variable\n", trgDevNm)
		}
	}

	// scoring system has 1 point for discrete and 1 for non-gl backend
	hiscore := 0
	best := 0
	for gi := range n {
		score := 0
		props := gpus[gi].GetInfo()
		if gpuIsBadBackend(props.BackendType) {
			continue
		}
		if props.AdapterType == wgpu.AdapterTypeDiscreteGPU {
			vnm := actualVendorName(&props)
			if (runtime.GOOS == "linux" || runtime.GOOS == "windows") && vnm == "nvidia" {
				if Debug || DebugAdapter {
					fmt.Println("not selecting discrete nvidia GPU: tends to crash when resizing windows")
				}
				score--
			} else {
				score++
			}
		}
		if !gpuIsGLdBackend(props.BackendType) {
			score++
		}
		if score > hiscore {
			hiscore = score
			best = gi
		}
	}
	return best
}

func (gp *GPU) SelectComputeGPU(gpus []*wgpu.Adapter) int {
	n := len(gpus)
	if n == 1 {
		return 0
	}

	var discrete []int
	for gi := range n {
		props := gpus[gi].GetInfo()
		if gpuIsBadBackend(props.BackendType) {
			continue
		}
		if props.AdapterType == wgpu.AdapterTypeDiscreteGPU {
			discrete = append(discrete, gi)
		}
	}

	ndisc := len(discrete)

	if ndisc > 0 && SelectAdapter >= 0 && SelectAdapter < ndisc {
		return discrete[SelectAdapter]
	}
	trgDevNm := ""
	if ev := os.Getenv("GPU_DEVICE"); ev != "" {
		trgDevNm = ev
	}
	if gp.ComputeOnly {
		if ev := os.Getenv("GPU_COMPUTE_DEVICE"); ev != "" {
			trgDevNm = ev
		}
	}

	if trgDevNm != "" {
		idx, err := strconv.Atoi(trgDevNm)
		if err == nil && idx >= 0 && idx < ndisc {
			return discrete[idx]
		}
		if err == nil && idx >= 0 && idx < n {
			return idx
		}
		for gi := range n {
			props := gpus[gi].GetInfo()
			if gpuIsBadBackend(props.BackendType) {
				continue
			}
			pnm := adapterName(&props)
			if strings.Contains(pnm, trgDevNm) {
				devNm := props.Name
				if Debug {
					log.Printf("gpu: selected device named: %s, specified in GPU_DEVICE or GPU_COMPUTE_DEVICE environment variable, index: %d\n", devNm, gi)
				}
				return gi
			}
		}
		if Debug {
			log.Printf("gpu: unable to find device named: %s, specified in GPU_DEVICE or GPU_COMPUTE_DEVICE environment variable\n", trgDevNm)
		}
	}

	// scoring system has 1 point for discrete and 1 for non-gl backend
	hiscore := 0
	best := 0
	for gi := range n {
		score := 0
		props := gpus[gi].GetInfo()
		if gpuIsBadBackend(props.BackendType) {
			continue
		}
		if props.AdapterType == wgpu.AdapterTypeDiscreteGPU {
			score++
		}
		if !gpuIsGLdBackend(props.BackendType) {
			score++
		}
		if score > hiscore {
			hiscore = score
			best = gi
		}
	}
	return best
}

func gpuIsGLdBackend(bet wgpu.BackendType) bool {
	return bet == wgpu.BackendTypeOpenGL || bet == wgpu.BackendTypeOpenGLES
}

func gpuIsBadBackend(bet wgpu.BackendType) bool {
	return bet == wgpu.BackendTypeUndefined || bet == wgpu.BackendTypeNull
}

// Release releases GPU resources -- call after everything else has been destroyed
func (gp *GPU) Release() {
	if gp.GPU != nil {
		gp.GPU.Release()
		gp.GPU = nil
	}
}

// NewDevice returns a new device for given GPU.
// It gets the Queue for this device.
func (gp *GPU) NewDevice() (*Device, error) {
	return NewDevice(gp)
}

// PropertiesString returns a human-readable summary of the GPU properties.
func (gp *GPU) PropertiesString() string {
	return "\n######## GPU Properties\n" + reflectx.StringJSON(&gp.Properties) + reflectx.StringJSON(gp.Limits.Limits)
}

// NoDisplayGPU Initializes WebGPU and returns that and a new
// GPU device, without using an existing surface window.
func NoDisplayGPU() (*GPU, *Device, error) {
	gp := NewGPU(nil)
	dev, err := NewDevice(gp)
	return gp, dev, err
}
