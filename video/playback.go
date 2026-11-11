// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package video

import (
	"sync"
	"time"

	"cogentcore.org/core/base/errors"
)

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

	// Muted is whether the audio is not heard (see [Playback.SetMuted]).
	Muted bool

	// Volume is the volume of the audio, from 0 to 1, where the zero value
	// is full volume (see [Playback.SetVolume]).
	Volume float32

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

	// audio is the audio being played, if any.
	audio *audio

	// position is the time in the media of the frame shown last.
	position time.Duration
}

// Play starts playing the media from the start, stopping any playing
// before. The media is decoded in the background, so this returns after
// opening it.
func (p *Playback) Play() error {
	return p.PlayFrom(0)
}

// PlayFrom starts playing the media from the given time, stopping any
// playing before, as for [Playback.Play].
func (p *Playback) PlayFrom(t time.Duration) error {
	return p.playFrom(t, false)
}

// playFrom plays the media from the given time, paused if paused.
func (p *Playback) playFrom(t time.Duration, paused bool) error {
	if p.Media == nil {
		return errors.New("video: no media to play")
	}
	p.Stop()
	p.mu.Lock()
	done := make(chan struct{})
	p.done, p.paused, p.position = done, paused, t
	if paused {
		p.pausedAt, p.resume = time.Now(), make(chan struct{})
	}
	p.mu.Unlock()
	frames, samples, errs, err := p.start(t, done)
	if err != nil {
		p.Stop() // (not playing)
		return err
	}
	go p.run(done, frames, samples, errs)
	return nil
}

// Seek plays the media from the given time (within its duration), keeping
// it paused if it is paused, which then shows the frame at that time.
func (p *Playback) Seek(t time.Duration) error {
	if d := p.Media.Duration; d > 0 && t > d {
		t = d
	}
	return p.playFrom(max(t, 0), p.IsPaused())
}

// Position returns the time in the media of the frame shown last (or the
// time that it is played from, before its first frame is shown).
func (p *Playback) Position() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.position
}

// SetMuted sets whether the audio is not heard, while it plays too.
func (p *Playback) SetMuted(muted bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Muted = muted
	if p.audio != nil {
		p.audio.setVolume(p.Volume, p.Muted)
	}
}

// SetVolume sets the volume of the audio, from 0 to 1, while it plays too.
func (p *Playback) SetVolume(volume float32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Volume = min(max(volume, 0), 1)
	if p.Volume == 0 { // (0 is full volume as the zero value)
		p.Volume = 1e-6
	}
	if p.audio != nil {
		p.audio.setVolume(p.Volume, p.Muted)
	}
}

// start starts decoding the media from the given time, and playing its
// audio, if any (also when it is muted, so that it can be heard later).
func (p *Playback) start(t time.Duration, done chan struct{}) (<-chan Frame, <-chan [2]float64, <-chan error, error) {
	hasAudio := p.Media.HasAudio && audioAvailable()
	frames, samples, errs, err := p.Media.DecodeFrom(t, p.Width, p.Height, hasAudio, done)
	if err != nil || !hasAudio {
		return frames, samples, errs, err
	}
	p.mu.Lock()
	p.audio = playAudio(samples, p.paused, p.Volume, p.Muted)
	p.mu.Unlock()
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
			if frames, samples, errs, err = p.start(0, done); err != nil {
				p.error(err)
				return
			}
			base, firstFrame = last, true
			continue
		}
		// The first frame (as of a seek) is shown at once, also while
		// paused, and the others at their times after it.
		showNow := firstFrame && base == 0 // (not again in a loop)
		if firstFrame {
			first, firstFrame = f.Time, false
			if showNow { // the time of the playing starts now
				start = p.restart()
			}
		}
		at := base + f.Time - first
		last = at + p.frameTime()

		// Wait for its time, waiting longer while paused.
		for !showNow {
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
		p.mu.Lock()
		p.position = f.Time
		p.mu.Unlock()
		p.OnFrame(f)
	}
}

// frameTime returns the duration of a frame of the media.
func (p *Playback) frameTime() time.Duration {
	if fr := p.Media.FrameRate; fr > 0 {
		return time.Duration(float64(time.Second) / fr)
	}
	return time.Second / 30
}

// restart returns the time now as the start of the playing, from which
// the time paused is counted again.
func (p *Playback) restart() time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	p.pauseShift = 0
	if p.paused {
		p.pausedAt = now
	}
	return now
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
	if p.audio != nil {
		p.audio.setPaused(paused)
	}
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
