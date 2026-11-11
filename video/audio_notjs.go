// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js && !novideo

package video

import (
	"sync"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
)

// SpeakerSampleRate is the sample rate of the speaker.
const SpeakerSampleRate beep.SampleRate = sampleRate

// speakerInit initializes the speaker once, for all playbacks.
var speakerInit = sync.OnceValue(func() error {
	return speaker.Init(SpeakerSampleRate, SpeakerSampleRate.N(time.Second/10))
})

// audioAvailable returns whether audio can be played.
func audioAvailable() bool { return speakerInit() == nil }

// audio is the audio of a playing on the speaker.
type audio struct {
	ctrl *beep.Ctrl
}

// playAudio starts playing the given audio samples on the speaker,
// paused if paused.
func playAudio(samples <-chan [2]float64, paused bool) *audio {
	a := &audio{ctrl: &beep.Ctrl{Streamer: streamSamples(samples), Paused: paused}}
	speaker.Play(a.ctrl)
	return a
}

// setPaused pauses or resumes the audio.
func (a *audio) setPaused(paused bool) {
	speaker.Lock()
	a.ctrl.Paused = paused
	speaker.Unlock()
}

// streamSamples returns a streamer of the audio samples from the given
// channel. It does not wait for samples that are not decoded yet (which
// would stop all of the sounds of the speaker), playing silence instead,
// and it ends when the channel is closed.
func streamSamples(samples <-chan [2]float64) beep.Streamer {
	return beep.StreamerFunc(func(out [][2]float64) (int, bool) {
		for i := range out {
			select {
			case s, ok := <-samples:
				if !ok {
					return i, i > 0
				}
				out[i] = s
			default:
				out[i] = [2]float64{}
			}
		}
		return len(out), true
	})
}
