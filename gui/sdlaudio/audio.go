// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package sdlaudio

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jetsetilly/gopher2600/hardware/television/signal"
	"github.com/jetsetilly/gopher2600/hardware/television/specification"
	"github.com/jetsetilly/gopher2600/hardware/tia/audio"
	"github.com/jetsetilly/gopher2600/hardware/tia/audio/mix"
	"github.com/jetsetilly/gopher2600/logger"

	"github.com/veandco/go-sdl2/sdl"
)

// Audio outputs sound using SDL.
type Audio struct {
	id   sdl.AudioDeviceID
	spec sdl.AudioSpec

	buffer   []uint8
	bufferCt int

	// stereo buffers are used to mix a stereo output
	stereoCh0Buffer []uint8
	stereoCh1Buffer []uint8

	// audio preferences
	Prefs *Preferences

	// local copy of some oft used preference values (too expensive to access
	// for the preferences system every time SetAudio() is called) . we'll
	// update these on every call to queueAudio()
	stereo     bool
	discrete   bool
	separation int

	// whether the device is muted
	muted bool

	// SetSpec() control. we don't want the sample frequency of the driver to
	// change too often so we throttle it by forwarding the request to an
	// interim go routine that either:
	// 1) waits for a timeout and forwards the change to the main thread
	// 2) waits for a cancel. cancels happen when a new change request is made
	//       before the timeout
	updateSync   sync.WaitGroup
	updateCancel chan bool
	updateCommit chan specification.Spec
	updateID     string

	// the most recent specification ID to be commited
	committedID string

	// measure size of audio queue periodically and cull it if it's getting too
	// long. called from SetAudio() so it is only checked when the emulation is
	// running
	queuedBytesMeasure *time.Ticker
	QueuedBytes        atomic.Int32
}

const (
	stretchAmount   = 3
	bufferLen       = 4 * stretchAmount // 4 bytes per sample
	stereoBufferLen = 1024
)

// NewAudio is the preferred method of initialisation for the Audio Type.
func NewAudio() (*Audio, error) {
	aud := &Audio{
		buffer:             make([]uint8, bufferLen),
		stereoCh0Buffer:    make([]uint8, stereoBufferLen),
		stereoCh1Buffer:    make([]uint8, stereoBufferLen),
		updateCancel:       make(chan bool),
		updateCommit:       make(chan specification.Spec),
		queuedBytesMeasure: time.NewTicker(250 * time.Millisecond),
	}

	var err error

	aud.Prefs, err = NewPreferences()
	if err != nil {
		return nil, fmt.Errorf("sdlaudio: %w", err)
	}
	aud.stereo = aud.Prefs.Stereo.Get().(bool)
	aud.discrete = aud.Prefs.Discrete.Get().(bool)
	aud.separation = aud.Prefs.Separation.Get().(int)

	aud.setSpec(specification.SpecNTSC)

	logger.Logf(logger.Allow, "sdlaudio", "id: %d", aud.id)
	logger.Logf(logger.Allow, "sdlaudio", "format: %d", aud.spec.Format)
	logger.Logf(logger.Allow, "sdlaudio", "channels: %d", aud.spec.Channels)
	logger.Logf(logger.Allow, "sdlaudio", "silence: %d", aud.spec.Silence)

	return aud, nil
}

// EndMixing implements the television.AudioMixer interface.
func (aud *Audio) EndMixing() error {
	if aud.id == 0 {
		return nil
	}
	sdl.CloseAudioDevice(aud.id)
	aud.id = 0
	return nil
}

// Reset implements the television.AudioMixer interface.
func (aud *Audio) Reset() {
	if aud.id == 0 {
		return
	}

	// fill buffers with silence
	for i := range aud.buffer {
		aud.buffer[i] = aud.spec.Silence
	}
	for i := range aud.stereoCh0Buffer {
		aud.stereoCh0Buffer[i] = aud.spec.Silence
		aud.stereoCh1Buffer[i] = aud.spec.Silence
	}

	sdl.ClearQueuedAudio(aud.id)
}

// SetSpec implements the television.RealtimeAudioMixer interface.
func (aud *Audio) SetSpec(spec specification.Spec) {
	if spec.ID == aud.updateID || spec.ID == aud.committedID {
		return
	}

	// record the request. if the same values are requested again we ignore it
	aud.updateID = spec.ID

	// cancel any outstanding requests
	select {
	case aud.updateCancel <- true:
	default:
	}

	// drain commit channel
	select {
	case <-aud.updateCommit:
	default:
	}

	// wait for any cancelled/ongoing request to complete
	aud.updateSync.Wait()

	// drain cancel channel in case we sent a cancel request when there was no
	// outstanding request
	select {
	case <-aud.updateCancel:
	default:
	}

	// start new request
	aud.updateSync.Add(1)

	go func() {
		// request always signals when it's done
		defer aud.updateSync.Done()

		// wait for cancel request or a timeout
		select {
		case <-aud.updateCancel:
			return
		case <-time.After(100 * time.Millisecond):
			// if we see the timeout signal then commit the request to the main
			// audio goroutine
			aud.updateCommit <- spec
		}
	}()
}

func (aud *Audio) setSpec(spec specification.Spec) {
	// this check is different to the one in the public SetSpec() function. this
	// check protects against instances of brief refresh rate interruption.
	// without this check the final updateRequest will get through and we
	// reinitisaliase the audio device with the same settings, creating a
	// audible gap in the sound
	//
	// we could put this check in the SetRefreshRate() function alongside the
	// other check but this is clearer and means any other codepath to this
	// function is covered.
	//
	// also, rather than scanlines and refreshRate, we could compare against the
	// calculated sample frequency that is to used for reinitialisation. this
	// would cover the instance where the calculation arrives at the same
	// frequency value through different inputs. I've not tested if that can
	// happen but it seems unlikely
	if spec.ID == aud.committedID {
		return
	}
	aud.committedID = spec.ID

	if aud.id > 0 {
		sdl.ClearQueuedAudio(aud.id)
		sdl.CloseAudioDevice(aud.id)
	}

	sampleFreq := float32(spec.ScanlinesTotal) * float32(spec.RefreshRate) * audio.SamplesPerScanline
	logger.Logf(logger.Allow, "sdlaudio", "calculated frequency: %d * %.2f * %d = %.2f",
		spec.ScanlinesTotal, spec.RefreshRate, audio.SamplesPerScanline, sampleFreq)

	request := &sdl.AudioSpec{
		Freq:     int32(sampleFreq),
		Format:   sdl.AUDIO_S16MSB,
		Channels: 2,
		Samples:  uint16(len(aud.buffer)),
	}

	var err error
	var actual sdl.AudioSpec

	aud.id, err = sdl.OpenAudioDevice("", false, request, &actual, 0)
	if err != nil {
		logger.Log(logger.Allow, "sdlaudio", err.Error())
	}
	aud.spec = actual

	logger.Logf(logger.Allow, "sdlaudio", "requested frequency: %d samples/sec", int(sampleFreq))
	logger.Logf(logger.Allow, "sdlaudio", "actual frequency: %d samples/sec", aud.spec.Freq)
	logger.Logf(logger.Allow, "sdlaudio", "buffer size: %d samples", len(aud.buffer))

	aud.Reset()
	sdl.PauseAudioDevice(aud.id, aud.muted)
}

// Mute silences the audio device.
func (aud *Audio) Mute(muted bool) {
	if aud.id == 0 {
		return
	}
	sdl.ClearQueuedAudio(aud.id)
	sdl.PauseAudioDevice(aud.id, muted)
	aud.muted = muted
}

const (
	rateRepeat  = 1000
	rateStretch = 2000
	rateDrop    = 10000
	rateReset   = 20000

	QueueOkay    = 2000
	QueueWarning = 8000
)

// SetAudio implements the television.AudioMixer interface.
func (aud *Audio) SetAudio(sig []signal.AudioSignalAttributes) error {
	if len(sig) > specification.AbsoluteMaxScanlines*audio.SamplesPerScanline {
		panic("too many samples received by SetAudio() in one call")
	}

	if aud.id == 0 {
		return nil
	}

	// resolve specifcation committal
	select {
	case u := <-aud.updateCommit:
		aud.setSpec(u)
	default:
	}

	// always measure queued bytes
	b := int32(sdl.GetQueuedAudioSize(aud.id))

	// update queuedBytes value less frequently
	select {
	case <-aud.queuedBytesMeasure.C:
		aud.QueuedBytes.Store(b)
	default:
		// update queuedBytes immediately if it's running low
		if b < rateRepeat {
			aud.QueuedBytes.Store(b)
		}
	}

	// regulate rate by either flushing the queue (which we only do as a last
	// resort) or by dropping the latest batch of samples
	if b > rateReset {
		logger.Logf(logger.Allow, "sdlaudio", "flushed audio queue: %d", b)
		sdl.ClearQueuedAudio(aud.id)
		aud.QueuedBytes.Store(int32(sdl.GetQueuedAudioSize(aud.id)))
	} else if b > rateDrop {
		// drop samples. the number of samples is so small that we won't audibly miss them
		return nil
	}

	// we still want to measure the audio queue even if the audio is muted. so
	// this check comes after the measurement
	if aud.muted {
		return nil
	}

	// update local preference values. we don't need these values if the audio
	// is muted
	aud.stereo = aud.Prefs.Stereo.Get().(bool)
	aud.discrete = aud.Prefs.Discrete.Get().(bool)
	aud.separation = aud.Prefs.Separation.Get().(int)

	// stretch each sample as required
	stretch := 1
	if b < rateStretch {
		stretch = stretchAmount
	}

	for _, s := range sig {
		v0 := s.AudioChannel0
		v1 := s.AudioChannel1

		aud.stereoCh0Buffer = aud.stereoCh0Buffer[1:]
		aud.stereoCh0Buffer = append(aud.stereoCh0Buffer, v0)
		aud.stereoCh1Buffer = aud.stereoCh1Buffer[1:]
		aud.stereoCh1Buffer = append(aud.stereoCh1Buffer, v1)

		if aud.stereo {
			var s0, s1 int16

			if aud.discrete {
				// discrete stereo channels
				s0, s1 = mix.Stereo(v0, v1)
			} else {
				// reverb mix
				var idx int
				switch aud.separation {
				case 1:
					idx = stereoBufferLen - 256
				case 2:
					idx = stereoBufferLen - 512
				case 3:
					idx = 0
				default:
					idx = stereoBufferLen
				}
				s0, s1 = mix.Stereo(v0+(aud.stereoCh1Buffer[idx]>>1), v1+(aud.stereoCh0Buffer[idx]>>1))
			}

			for range stretch {
				aud.buffer[aud.bufferCt] = uint8(s0>>8) + aud.spec.Silence
				aud.bufferCt++
				aud.buffer[aud.bufferCt] = uint8(s0) + aud.spec.Silence
				aud.bufferCt++
				aud.buffer[aud.bufferCt] = uint8(s1>>8) + aud.spec.Silence
				aud.bufferCt++
				aud.buffer[aud.bufferCt] = uint8(s1) + aud.spec.Silence
				aud.bufferCt++
			}
		} else {
			m := mix.Mono(v0, v1)
			for range stretch {
				aud.buffer[aud.bufferCt] = uint8(m>>8) + aud.spec.Silence
				aud.bufferCt++
				aud.buffer[aud.bufferCt] = uint8(m) + aud.spec.Silence
				aud.bufferCt++
				aud.buffer[aud.bufferCt] = uint8(m>>8) + aud.spec.Silence
				aud.bufferCt++
				aud.buffer[aud.bufferCt] = uint8(m) + aud.spec.Silence
				aud.bufferCt++
			}
		}

		// we don't want to overrun the buffer but we don't want to call queue() too often
		if aud.bufferCt >= len(aud.buffer) {
			err := aud.queue(false)
			if err != nil {
				return err
			}
		}
	}

	// make sure we've added something
	return aud.queue(true)
}

func (aud *Audio) queue(allowRepeat bool) error {
	if aud.bufferCt == 0 {
		return nil
	}

	// we always add to the queue at least once. if it results in too much data
	// then we regulate that on the next call to SetAudio()
	if err := sdl.QueueAudio(aud.id, aud.buffer[:aud.bufferCt]); err != nil {
		return fmt.Errorf("sdlaudio: %w", err)
	}

	if allowRepeat {
		b := int(sdl.GetQueuedAudioSize(aud.id))

		// make sure we have a minimum amount of data in the queue
		for b < rateRepeat {
			if err := sdl.QueueAudio(aud.id, aud.buffer[:aud.bufferCt]); err != nil {
				return fmt.Errorf("sdlaudio: %w", err)
			}
			b = int(sdl.GetQueuedAudioSize(aud.id))
		}
	}

	// the data in the buffer is now dealt with
	aud.bufferCt = 0

	return nil
}
