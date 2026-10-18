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
	// samples rendered at a time. each chunk is passed to the television (for
	// video recording) and, if there is any sound in it, to the audio device
	speakJetBufferLen = 128

	// output gain. the chip's output peaks at about 0.5
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

// NewSpeakJet creates an engine that emulates the SpeakJet chip. it runs in
// emulation time: the AtariVox steps it every CPU cycle (AtariVoxClocked)
func NewSpeakJet(env *environment.Environment) (AtariVoxEngine, error) {
	err := env.TV.OpenAuxAudio(speakjet.SampleRate, speakJetBufferLen)
	if err != nil {
		return nil, err
	}

	sj := &speakJetEngine{
		env:  env,
		chip: speakjet.NewSpeakJet(),
	}
	sj.setClock()

	return sj, nil
}

// the CPU clock follows the television specification, which can change while
// the emulation runs. 76 CPU cycles per scanline
func (sj *speakJetEngine) setClock() {
	sj.cpuClock = int(sj.env.TV.GetFrameInfo().Spec.HorizontalScanRate * 76)
}

// Step implements the AtariVoxClocked interface.
func (sj *speakJetEngine) Step() {
	sj.clock += speakjet.SampleRate
	if sj.clock < sj.cpuClock {
		return
	}
	sj.clock -= sj.cpuClock

	sample := sj.buffer[sj.bufferIdx : sj.bufferIdx+1]
	_ = sj.chip.Render(sample)
	sample[0] = max(-1, min(1, sample[0]*speakJetGain))

	sj.bufferIdx++
	if sj.bufferIdx == speakJetBufferLen {
		sj.endChunk()
	}
}

func (sj *speakJetEngine) endChunk() {
	sj.env.TV.PushAuxAudio(sj.buffer[:], speakjet.SampleRate)
	sj.bufferIdx = 0
	sj.setClock()
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

// Flush implements the AtariVoxEngine interface. the SpeakJet speaks as the
// bytes arrive so there is nothing to flush
func (sj *speakJetEngine) Flush() {
}

// BufferHalfFull implements the AtariVoxStatus interface.
func (sj *speakJetEngine) BufferHalfFull() bool {
	return sj.chip.BufferHalfFull()
}
