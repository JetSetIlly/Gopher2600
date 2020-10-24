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
)

// tag string used in called to Log().
const soundloadLogTag = "supercharger: soundload"

// SoundLoad implements the Tape interface. It loads data from a sound file.
//
// Compared to FastLoad this method is more 'authentic' and uses the BIOS
// correctly.
type SoundLoad struct {
	cart *Supercharger

	// sample levels
	samples []float32

	// spped of samples in Hz
	sampleRate float64

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
}

// newSoundLoad is the preferred method of initialisation for the SoundLoad type.
func newSoundLoad(cart *Supercharger, loader cartridgeloader.Loader) (tape, error) {
	tap := &SoundLoad{
		cart: cart,
	}

	// get PCM data from data loaded from file
	pcm, err := getPCM(loader)
	if err != nil {
		return nil, fmt.Errorf("soundload: %v", err)
	}

	numChannels := pcm.Format.NumChannels
	tap.sampleRate = float64(pcm.Format.SampleRate)

	// copy just one channel worth of bits
	tap.samples = make([]float32, 0, len(pcm.Data)/numChannels)
	for i := 0; i < len(pcm.Data); i += numChannels {
		tap.samples = append(tap.samples, pcm.Data[i])
	}

	// PCM info
	logger.Log(soundloadLogTag, fmt.Sprintf("num channels: %d (using one)", numChannels))
	logger.Log(soundloadLogTag, fmt.Sprintf("sample rate: %0.2fHz", tap.sampleRate))
	logger.Log(soundloadLogTag, fmt.Sprintf("total time: %.02fs", float64(len(tap.samples))/tap.sampleRate))

	// the length of time of each sample in microseconds
	timePerSample := 1000000.0 / tap.sampleRate
	logger.Log(soundloadLogTag, fmt.Sprintf("time per sample: %.02fus", timePerSample))

	// number of samples in a cycle for it to be interpreted as a zero or a one
	// values taken from "Atari 2600 Mappers" document by Kevin Horton
	logger.Log(soundloadLogTag, fmt.Sprintf("min/opt/max samples for zero-bit: %d/%d/%d",
		int(158.0/timePerSample), int(227.0/timePerSample), int(317.0/timePerSample)))
	logger.Log(soundloadLogTag, fmt.Sprintf("min/opt/max samples for one-bit: %d/%d/%d",
		int(317.0/timePerSample), int(340.0/timePerSample), int(2450.0/timePerSample)))

	// calculate tape regulator speed. 1190000 is the frequency at which step() is called (1.19MHz)
	tap.regulator = int(math.Round(1190000.0 / tap.sampleRate))
	logger.Log(soundloadLogTag, fmt.Sprintf("tape regulator: %d", tap.regulator))

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
	if !tap.playing {
		if tap.playDelay < 30000 {
			tap.playDelay++
			return 0x00, nil
		}
		tap.playing = true
		tap.playDelay = 0
	}

	if tap.samples[tap.idx] > 0.0 {
		return 0x01, nil
	}
	return 0x00, nil
}

// step implements the Tape interface.
func (tap *SoundLoad) step() {
	if !tap.playing {
		return
	}

	if tap.regulatorCt <= tap.regulator {
		tap.regulatorCt++
		return
	}
	tap.regulatorCt = 0

	// make sure we don't try to read past end of tape
	if tap.idx >= len(tap.samples)-1 {
		tap.playing = false
		return
	}
	tap.idx++
}

// Rewind implements the mapper.CartTapeBus interface.
func (tap *SoundLoad) Rewind() bool {
	// rewinding happens instantaneously
	tap.idx = 0
	logger.Log(soundloadLogTag, "tape rewound")
	return true
}

// SetTapeCounter implements the mapper.CartTapeBus interface.
func (tap *SoundLoad) SetTapeCounter(c int) {
	if c >= len(tap.samples) {
		c = len(tap.samples)
	}
	tap.idx = c
}

// the number of samples to copy and return from GetTapeState().
const numStateSamples = 100

func (tap *SoundLoad) GetTapeState() (bool, mapper.CartTapeState) {
	state := mapper.CartTapeState{
		Counter:    tap.idx,
		MaxCounter: len(tap.samples),
		Time:       float64(tap.idx) / tap.sampleRate,
		MaxTime:    float64(len(tap.samples)) / tap.sampleRate,
		Data:       make([]float32, numStateSamples),
	}

	if tap.idx < len(tap.samples) {
		if tap.idx > len(tap.samples)-numStateSamples {
			copy(state.Data, tap.samples[tap.idx:])
		} else {
			copy(state.Data, tap.samples[tap.idx:tap.idx+numStateSamples])
		}
	}

	return true, state
}
