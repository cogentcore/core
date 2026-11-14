// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !js && !novideo

package video

import (
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
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

// audio is the audio of a playing on the speaker, or, without a speaker
// (as when the system has no audio device), of a silent playing that
// consumes the samples at their rate, which times the playing of media
// that has only audio (see [Playback]).
type audio struct {
	ctrl   *beep.Ctrl
	volume *effects.Volume

	// played is the number of samples played (not of the silence played
	// while the samples are not decoded yet, as of a stream over the
	// network), which is the time played.
	played atomic.Int64

	// ended is closed when all of the samples have been played.
	ended     chan struct{}
	endedOnce sync.Once

	// silent is whether it is played without a speaker, paused when paused
	// and ending when stopped (with mu).
	silent          bool
	mu              sync.Mutex
	paused, stopped bool
}

// playAudio starts playing the given audio samples on the speaker,
// paused if paused, at the given volume (see [Playback.Volume]), or muted.
// Without a speaker, the samples are played silently, at their rate.
func playAudio(samples <-chan [2]float64, paused bool, volume float32, muted bool) *audio {
	a := &audio{ended: make(chan struct{})}
	if !audioAvailable() {
		a.silent, a.paused = true, paused
		go a.playSilent(samples)
		return a
	}
	a.ctrl = &beep.Ctrl{Streamer: a.streamSamples(samples), Paused: paused}
	a.volume = &effects.Volume{Streamer: a.ctrl, Base: 2}
	a.setVolumeLocked(volume, muted)
	speaker.Play(a.volume)
	return a
}

// setVolume sets the volume of the audio (see [Playback.Volume]), or mutes it.
func (a *audio) setVolume(volume float32, muted bool) {
	if a.silent {
		return
	}
	speaker.Lock()
	a.setVolumeLocked(volume, muted)
	speaker.Unlock()
}

// setVolumeLocked sets the volume, with the lock of the speaker held, or
// before it plays. The volume of beep is in powers of its base (2): a
// volume of 1/2 is -1.
func (a *audio) setVolumeLocked(volume float32, muted bool) {
	if volume == 0 {
		volume = 1
	}
	a.volume.Silent = muted || volume < 1e-3
	a.volume.Volume = math.Log2(float64(volume))
}

// setPaused pauses or resumes the audio.
func (a *audio) setPaused(paused bool) {
	if a.silent {
		a.mu.Lock()
		a.paused = paused
		a.mu.Unlock()
		return
	}
	speaker.Lock()
	a.ctrl.Paused = paused
	speaker.Unlock()
}

// stop stops the audio for good, removing it from the speaker (a paused
// streamer would stay in its mixer).
func (a *audio) stop() {
	if a.silent {
		a.mu.Lock()
		a.stopped = true
		a.mu.Unlock()
		return
	}
	speaker.Lock()
	a.ctrl.Streamer = nil // (a Ctrl without a streamer is drained)
	speaker.Unlock()
}

// time returns the time of the audio played.
func (a *audio) time() time.Duration {
	return time.Duration(a.played.Load()) * time.Second / sampleRate
}

// end marks the audio as having played all of its samples.
func (a *audio) end() {
	a.endedOnce.Do(func() { close(a.ended) })
}

// streamSamples returns a streamer of the audio samples from the given
// channel. It does not wait for samples that are not decoded yet (which
// would stop all of the sounds of the speaker), playing silence instead,
// and it ends when the channel is closed.
func (a *audio) streamSamples(samples <-chan [2]float64) beep.Streamer {
	return beep.StreamerFunc(func(out [][2]float64) (int, bool) {
		n := 0
		defer func() { a.played.Add(int64(n)) }()
		for i := range out {
			select {
			case s, ok := <-samples:
				if !ok {
					a.end()
					return i, i > 0
				}
				out[i] = s
				n++
			default:
				out[i] = [2]float64{}
			}
		}
		return len(out), true
	})
}

// playSilent consumes the given samples at their rate while not paused,
// without a speaker, until the channel is closed.
func (a *audio) playSilent(samples <-chan [2]float64) {
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	last := time.Now()
	owed := 0.0 // the samples due to be played
	for now := range tick.C {
		a.mu.Lock()
		paused, stopped := a.paused, a.stopped
		a.mu.Unlock()
		if stopped {
			return
		}
		if !paused {
			owed += now.Sub(last).Seconds() * sampleRate
		}
		last = now
	consume:
		for ; owed >= 1; owed-- {
			select {
			case _, ok := <-samples:
				if !ok {
					a.end()
					return
				}
				a.played.Add(1)
			default: // (not decoded yet: played as silence)
				owed = 0
				break consume
			}
		}
	}
}
