// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package video

import (
	"sync"
	"time"

	"cogentcore.org/core/base/errors"
	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
)

const (
	// sampleRate is the rate of the audio samples played by the speaker.
	sampleRate = 44100

	// SpeakerSampleRate is the sample rate of the speaker.
	SpeakerSampleRate beep.SampleRate = sampleRate
)

// speakerInit initializes the speaker once, for all playbacks.
var speakerInit = sync.OnceValue(func() error {
	return speaker.Init(SpeakerSampleRate, SpeakerSampleRate.N(time.Second/10))
})

// Playback plays a [Media], independent of any widget: it calls OnFrame
// with each video frame at its time, and plays the audio on the speaker.
// A [Video] shows a playback; other widgets (as an image of a web page)
// can show one too.
type Playback struct {

	// Media is the media that is played.
	Media *Media

	// Width and Height are the size to which the frames are scaled,
	// where 0 for a dimension keeps the aspect ratio (both 0 = the inherent
	// size of the video).
	Width, Height int

	// Muted is whether the audio is not played.
	Muted bool

	// Loop is whether to play the media again from the start at its end.
	Loop bool

	// OnFrame is called with each video frame at the time to show it,
	// in the goroutine of the playback.
	OnFrame func(f Frame)

	// OnEnd is called, if set, at the end of the media (unless Loop),
	// in the goroutine of the playback.
	OnEnd func()

	// OnError is called, if set, with an error decoding the media,
	// which stops the playing; it is logged otherwise.
	OnError func(err error)

	mu sync.Mutex

	// done is closed to stop the current playing.
	done chan struct{}

	// paused is whether the playing is paused, since pausedAt,
	// and resume is closed when it is resumed.
	paused   bool
	pausedAt time.Time
	resume   chan struct{}

	// pauseShift is the time paused that run has not added to the start
	// of the playing yet.
	pauseShift time.Duration

	// audio is the control of the audio being played, if any.
	audio *beep.Ctrl
}

// Play starts playing the media from the start, stopping any playing
// before. The media is decoded in the background, so this returns after
// opening it.
func (p *Playback) Play() error {
	if p.Media == nil {
		return errors.New("video: no media to play")
	}
	p.Stop()
	p.mu.Lock()
	done := make(chan struct{})
	p.done, p.paused = done, false
	p.mu.Unlock()
	frames, samples, errs, err := p.start(done)
	if err != nil {
		return err
	}
	go p.run(done, frames, samples, errs)
	return nil
}

// start starts decoding the media, and playing its audio, if any.
func (p *Playback) start(done chan struct{}) (<-chan Frame, <-chan [2]float64, <-chan error, error) {
	audio := !p.Muted && p.Media.HasAudio && speakerInit() == nil
	frames, samples, errs, err := p.Media.Decode(p.Width, p.Height, audio, done)
	if err != nil || !audio {
		return frames, samples, errs, err
	}
	ctrl := &beep.Ctrl{Streamer: streamSamples(samples)}
	p.mu.Lock()
	ctrl.Paused = p.paused
	p.audio = ctrl
	p.mu.Unlock()
	speaker.Play(ctrl)
	return frames, samples, errs, nil
}

// run shows the frames at their times, until the end of the media
// (playing it again for Loop), or until done is closed.
func (p *Playback) run(done chan struct{}, frames <-chan Frame, samples <-chan [2]float64, errs <-chan error) {
	start := time.Now()
	var first, base, last time.Duration // base: the start of this loop
	firstFrame := true
	for {
		// Get the next frame, at the end of the media playing it again.
		var f Frame
		var ok bool
		select {
		case f, ok = <-frames:
		case <-done:
			return
		}
		if !ok {
			if err := <-errs; err != nil {
				p.error(err)
				return
			}
			if !p.Loop {
				if p.OnEnd != nil {
					p.OnEnd()
				}
				return
			}
			var err error
			if frames, samples, errs, err = p.start(done); err != nil {
				p.error(err)
				return
			}
			base, firstFrame = last, true
			continue
		}
		if firstFrame {
			first, firstFrame = f.Time, false
		}
		at := base + f.Time - first
		last = at + time.Second/30

		// Wait for its time, waiting longer while paused.
		for {
			// (the start moves later by the time paused)
			start = p.startAfterPauses(start)
			p.mu.Lock()
			paused, resume := p.paused, p.resume
			p.mu.Unlock()
			if paused {
				select {
				case <-resume:
				case <-done:
					return
				}
				continue
			}
			wait := time.Until(start.Add(at))
			if wait <= 0 {
				break
			}
			select {
			case <-time.After(wait):
			case <-done:
				return
			}
		}
		select {
		case <-done:
			return
		default:
		}
		p.OnFrame(f)
	}
}

// startAfterPauses returns the given start of the playing moved later
// by the time paused since the last call.
func (p *Playback) startAfterPauses(start time.Time) time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	start = start.Add(p.pauseShift)
	p.pauseShift = 0
	return start
}

// error handles an error decoding the media.
func (p *Playback) error(err error) {
	if p.OnError != nil {
		p.OnError(err)
		return
	}
	errors.Log(err)
}

// Pause pauses the playing, if it is playing.
func (p *Playback) Pause() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.done == nil || p.paused {
		return
	}
	p.paused, p.pausedAt = true, time.Now()
	p.resume = make(chan struct{})
	p.setAudioPaused(true)
}

// Resume resumes the playing, if it is paused.
func (p *Playback) Resume() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.paused {
		return
	}
	p.paused = false
	p.pauseShift += time.Since(p.pausedAt)
	close(p.resume)
	p.setAudioPaused(false)
}

// setAudioPaused pauses or resumes the audio, with the lock held.
func (p *Playback) setAudioPaused(paused bool) {
	if p.audio == nil {
		return
	}
	speaker.Lock()
	p.audio.Paused = paused
	speaker.Unlock()
}

// IsPlaying returns whether the media is playing, and not paused.
func (p *Playback) IsPlaying() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.done != nil && !p.paused
}

// IsPaused returns whether the playing is paused.
func (p *Playback) IsPaused() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.paused
}

// Stop stops the playing, if any; Play plays the media from the start.
func (p *Playback) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.done != nil {
		close(p.done)
		p.done = nil
	}
	if p.paused {
		close(p.resume)
	}
	p.paused, p.pauseShift = false, 0
	p.setAudioPaused(true)
	p.audio = nil
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
