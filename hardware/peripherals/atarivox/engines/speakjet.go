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

package engines

import (
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/speakjet"
)

const (
	// the amount of data to buffer before pushing to the television
	speakJetBufferLen = 128

	// the volume gain
	speakJetGain = 1.8
)

type speakJetEngine struct {
	env  *environment.Environment
	chip *speakjet.SpeakJet

	// CPU cycles per second, and the count towards the next sample
	cpuClock int
	clock    int

	buffer    [speakJetBufferLen]float32
	bufferIdx int
}

// NewSpeakJet creates an engine that emulates the SpeakJet chip.
func NewSpeakJet(env *environment.Environment) (AtariVoxEngine, error) {
	err := env.TV.OpenAuxAudio(speakjet.SampleRate, speakJetBufferLen)
	if err != nil {
		return nil, err
	}

	sj := &speakJetEngine{
		env:  env,
		chip: speakjet.NewSpeakJet(),
	}
	sj.chip.Gain = speakJetGain
	sj.setClock()

	return sj, nil
}

func (sj *speakJetEngine) setClock() {
	// horizontal scan rate is approximately the same as the actual clock speed of the CPU
	sj.cpuClock = int(sj.env.TV.GetFrameInfo().Spec.HorizontalScanRate * 76)
}

// Step implements the AtariVoxRealtime interface.
func (sj *speakJetEngine) Step() {
	sj.clock += speakjet.SampleRate
	if sj.clock < sj.cpuClock {
		return
	}
	sj.clock -= sj.cpuClock

	sample := sj.buffer[sj.bufferIdx : sj.bufferIdx+1]
	_ = sj.chip.Render(sample)
	sample[0] = max(-1, min(1, sample[0]))

	sj.bufferIdx++
	if sj.bufferIdx == speakJetBufferLen {
		sj.env.TV.SetAuxAudio(sj.buffer[:], speakjet.SampleRate)
		sj.bufferIdx = 0
		sj.setClock()
	}
}

// Quit implements the AtariVoxEngine interface.
func (sj *speakJetEngine) Quit() {
	sj.env.TV.CloseAuxAudio()
}

// SpeakJet implements the AtariVoxEngine interface.
func (sj *speakJetEngine) SpeakJet(command uint8, data uint8) {
	if command >= 200 && command <= 254 {
		logger.Logf(sj.env, "speakjet", "sound effect: %#02x", command)
	}
	if !sj.chip.Write(command, data) {
		logger.Logf(sj.env, "speakjet", "buffer full: lost %#02x", command)
	}
}

// Flush implements the AtariVoxEngine interface.
func (sj *speakJetEngine) Flush() {
	// in the case of the SpeakJet implementation there is no need to flush
}

// BufferHalfFull implements the AtariVoxRealtime interface.
func (sj *speakJetEngine) BufferHalfFull() bool {
	return sj.chip.BufferHalfFull()
}
