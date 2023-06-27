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

package supercharger

import (
	"fmt"
	"math"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/notifications"
)

// Brief explanation of how the "tape" works:
//
// The tape "plays" and "stops" and "rewinds" automatically when the 6507 requires it.
//
// This is my current strategy:
//
// 1) Tape will play when address 1ff9 (or any of the cartridge mirrors) is
//    read. The RAM write bit must also be set for the tape to start playing.
//
// 2) Tape will stop when address fa1a (specifically) is read. This is the
//    point in the BIOS at which the program jumps to VCS RAM (at address
//    0x00fa) and the loaded program begins.
//
// 3) Rewinding: The "tape" is just a sound file (the emulator supports most
//    WAV and MP3 files) so "playing" is nothing more than keeping track of how
//    much of the file has been read and moving that progress pointer along
//    every CPU cycle. "Rewinding" therefore, is just a case of resetting the
//    progress pointer to zero when the end of the file has been reached.
//
// Eight bits of data is returned from the current tape position on every
// subsequent read of address 1ff9. Decoding of the audio signal is handled by
// the Supercharger BIOS.

// tag string used in called to Log().
const soundloadLogTag = "supercharger: soundload"

// SoundLoad implements the Tape interface. It loads data from a sound file.
//
// Compared to FastLoad this method is more 'authentic' and uses the BIOS
// correctly.
type SoundLoad struct {
	cart *Supercharger

	// sound data and format information
	pcm pcmData

	// current index of samples array
	idx int

	// is the tape currently playing
	playing bool

	// short delay before starting tape. allows the "rewind tape. press play"
	// message to be visible momentarily, rather than disconcertingly flashing
	// on and then off
	playDelay int

	// the speed at which the tape advances. the tape is advanced every call to step()
	// which happens at a rate of 3.57MHz.
	regulator   int
	regulatorCt int

	// stepLimiter is used to limit the number of times step() can be called
	// without load() being called and for the tape to continue "playing". once
	// stepLimiter reaches the stepLimit value, the tape stops
	stepLimiter int

	threshold float32
}

// newSoundLoad is the preferred method of initialisation for the SoundLoad type.
func newSoundLoad(cart *Supercharger, loader cartridgeloader.Loader) (tape, error) {
	tap := &SoundLoad{
		cart: cart,
	}

	var err error

	// get PCM data from data loaded from file
	tap.pcm, err = getPCM(loader)
	if err != nil {
		return nil, fmt.Errorf("soundload: %w", err)
	}

	if len(tap.pcm.data) == 0 {
		return nil, fmt.Errorf("soundload: no PCM data in file")
	}

	// the length of time of each sample in microseconds
	timePerSample := 1000000.0 / tap.pcm.sampleRate
	logger.Logf(soundloadLogTag, "time per sample: %.02fus", timePerSample)

	// number of samples in a cycle for it to be interpreted as a zero or a one
	// values taken from "Atari 2600 Mappers" document by Kevin Horton
	logger.Log(soundloadLogTag, fmt.Sprintf("min/opt/max samples for zero-bit: %d/%d/%d",
		int(158.0/timePerSample), int(227.0/timePerSample), int(317.0/timePerSample)))
	logger.Log(soundloadLogTag, fmt.Sprintf("min/opt/max samples for one-bit: %d/%d/%d",
		int(317.0/timePerSample), int(340.0/timePerSample), int(2450.0/timePerSample)))

	// calculate tape regulator speed. 1190000 is the frequency at which step() is called (1.19MHz)
	//
	// TODO: for non-NTSC machine the frequency step is called will be
	// different but it doesn't appear to have any effect on loading success so
	// we won't complicate the code by allowing the regulator to change
	tap.regulator = int(math.Round(1190000.0 / tap.pcm.sampleRate))
	logger.Logf(soundloadLogTag, "tape regulator: %d", tap.regulator)

	// threshold value is the average value in the PCM data
	var total float32
	for _, d := range tap.pcm.data {
		total += d
	}
	tap.threshold = total / float32(len(tap.pcm.data))

	// rewind tape to start of header
	tap.Rewind()

	return tap, nil
}

// snapshot implements the tape interface.
func (tap *SoundLoad) snapshot() tape {
	// not copying samples. each snapshot will point to the original array
	n := *tap
	return &n
}

// load implements the Tape interface.
func (tap *SoundLoad) load() (uint8, error) {
	tap.stepLimiter = 0
	if !tap.playing {
		if tap.playDelay < 30000 {
			tap.playDelay++
			return 0x00, nil
		}
		tap.cart.notificationHook(tap.cart, notifications.NotifySuperchargerSoundloadStarted)
		tap.playing = true
		tap.playDelay = 0
		logger.Log(soundloadLogTag, "tape playing")
	}

	if tap.pcm.data[tap.idx] > tap.threshold {
		return 0x01, nil
	}
	return 0x00, nil
}

// step implements the Tape interface.
func (tap *SoundLoad) step() {
	// auto-stop tape if load() has not been called "recently"
	const stepLimit = 100000
	if tap.stepLimiter < stepLimit {
		tap.stepLimiter++
		if tap.stepLimiter == stepLimit {
			logger.Log(soundloadLogTag, "tape stopped")
			tap.playing = false
		}
	}

	if !tap.playing {
		return
	}

	if tap.regulatorCt <= tap.regulator {
		tap.regulatorCt++
		return
	}
	tap.regulatorCt = 0

	// make sure we don't try to read past end of tape
	if tap.idx >= len(tap.pcm.data)-1 {
		tap.Rewind()
		return
	}
	tap.idx++
}

// Rewind implements the mapper.CartTapeBus interface.
func (tap *SoundLoad) Rewind() {
	// rewinding happens instantaneously
	tap.cart.notificationHook(tap.cart, notifications.NotifySuperchargerSoundloadRewind)
	tap.idx = 0
	logger.Log(soundloadLogTag, "tape rewound")
	tap.stepLimiter = 0
}

// SetTapeCounter implements the mapper.CartTapeBus interface.
func (tap *SoundLoad) SetTapeCounter(c int) {
	if c >= len(tap.pcm.data) {
		c = len(tap.pcm.data)
	}
	tap.idx = c
}

// the number of samples to copy and return from GetTapeState().
const numStateSamples = 100

func (tap *SoundLoad) GetTapeState() (bool, mapper.CartTapeState) {
	state := mapper.CartTapeState{
		Counter:    tap.idx,
		MaxCounter: len(tap.pcm.data),
		Time:       float64(tap.idx) / tap.pcm.sampleRate,
		MaxTime:    float64(len(tap.pcm.data)) / tap.pcm.sampleRate,
		Data:       make([]float32, numStateSamples),
	}

	if tap.idx < len(tap.pcm.data) {
		if tap.idx > len(tap.pcm.data)-numStateSamples {
			copy(state.Data, tap.pcm.data[tap.idx:])
		} else {
			copy(state.Data, tap.pcm.data[tap.idx:tap.idx+numStateSamples])
		}
	}

	return true, state
}
