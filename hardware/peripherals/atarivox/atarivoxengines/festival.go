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

package atarivoxengines

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/logger"
)

type festival struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser

	stream   []uint8
	phonemes strings.Builder

	quit chan bool
	say  chan string

	// some SpeakJet commands expect a following byte as a parameter. we don't
	// support these yet so we simply swallow them
	ignoreNextSpeakJetByte bool
}

// NewFestival creats a new festival instance and starts a new festival process
// which we'll communicate with via a stdin pipe.
func NewFestival(executablePath string) (AtariVoxEngine, error) {
	fest := &festival{
		quit: make(chan bool),
		say:  make(chan string),
	}

	cmd := exec.Command(executablePath)

	var err error

	fest.stdin, err = cmd.StdinPipe()
	if err != nil {
		return nil, curated.Errorf("festival: %s", err.Error())
	}

	fest.stdout, err = cmd.StdoutPipe()
	if err != nil {
		return nil, curated.Errorf("festival: %s", err.Error())
	}

	err = cmd.Start()
	if err != nil {
		return nil, curated.Errorf("festival: %s", err.Error())
	}

	go func() {
		for {
			select {
			case <-fest.quit:
				command := "(quit)"
				n, err := fest.stdin.Write([]byte(command))
				if n != len(command) || err != nil {
					logger.Logf("festival", "quit command doesn't appear to have succeeded")
				}
				return
			case text := <-fest.say:
				command := fmt.Sprintf("(SayPhones '(%s))", text)
				logger.Logf("festival", command)
				fest.stdin.Write([]byte(command))
			}
		}
	}()

	return fest, nil
}

// Quit implements the AtariVoxEngine interface.
func (fest *festival) Quit() {
	fest.quit <- true
}

// SpeakJet implements the AtariVoxEngine interface.
func (fest *festival) SpeakJet(b uint8) {
	if fest.ignoreNextSpeakJetByte {
		fest.ignoreNextSpeakJetByte = false
		return
	}

	// http://festvox.org/festvox-1.2/festvox_18.html

	// https://people.ece.cornell.edu/land/courses/ece4760/Speech/speakjetusermanual.pdf

	switch b {
	default:
		logger.Logf("festival", "unsupported byte (%d)", b)
		return

	case 0: // pause 0ms
		return
	case 1: // pause 100ms
		return
	case 2: // pause 200ms
		logger.Logf("festival", "pause: %d", b)
		fest.Flush()
		return
	case 3: // pause 700ms
		logger.Logf("festival", "pause: %d", b)
		fest.Flush()
		return
	case 4: // pause 30ms
		return
	case 5: // pause 60ms
		return
	case 6: // pause 90ms
		return

	case 7: // Fast
		logger.Logf("festival", "fast: not implemented")
		return
	case 8: // Slow
		logger.Logf("festival", "slow: not implemented")
		return
	case 14: // Stress
		logger.Logf("festival", "stress: not implemented")
		return
	case 15: // Relax
		logger.Logf("festival", "relax: not implemented")
		return
	case 20: // volume
		logger.Logf("festival", "volume: not implemented")
		fest.ignoreNextSpeakJetByte = true
		return
	case 21: // speed
		logger.Logf("festival", "speed: not implemented")
		fest.ignoreNextSpeakJetByte = true
		return
	case 22: // pitch
		logger.Logf("festival", "pitch: not implemented")
		fest.ignoreNextSpeakJetByte = true
		return
	case 23: // bend
		logger.Logf("festival", "bend: not implemented")
		fest.ignoreNextSpeakJetByte = true
		return
	case 24: // PortCtr
		logger.Logf("festival", "port ctr: not implemented")
		fest.ignoreNextSpeakJetByte = true
		return
	case 25: // Port
		logger.Logf("festival", "port: not implemented")
		fest.ignoreNextSpeakJetByte = true
		return
	case 26: // Repeat
		logger.Logf("festival", "repeat: not implemented")
		fest.ignoreNextSpeakJetByte = true
		return
	case 28: // Call Phrase
		logger.Logf("festival", "call phrase: not implemented")
		fest.ignoreNextSpeakJetByte = true
		return
	case 29: // Goto Phrase
		logger.Logf("festival", "goto phrase: not implemented")
		fest.ignoreNextSpeakJetByte = true
		return
	case 30: // Delay
		logger.Logf("festival", "delay: not implemented")
		fest.ignoreNextSpeakJetByte = true
		return
	case 31: // Reset
		fest.Flush()
		return

	case 128:
		fest.phonemes.WriteString("iy ")
	case 129:
		fest.phonemes.WriteString("ih ")

	case 130:
		fest.phonemes.WriteString("ey ")
	case 131:
		fest.phonemes.WriteString("eh ")
	case 132:
		fest.phonemes.WriteString("ae ")
	case 133:
		fest.phonemes.WriteString("o ") // cotton ??
	case 134:
		fest.phonemes.WriteString("uh ")
	case 135:
		fest.phonemes.WriteString("o ") // hot, clock, fox ??
	case 136:
		fest.phonemes.WriteString("aa ")
	case 137:
		fest.phonemes.WriteString("ow ")
	case 138:
		fest.phonemes.WriteString("uh ") // book, could, should ??  'ah' possibly)
	case 139:
		fest.phonemes.WriteString("uw ")

	case 140:
		fest.phonemes.WriteString("m ")
	case 141:
		fest.phonemes.WriteString("n ")
	case 142:
		fest.phonemes.WriteString("ow ")
	case 143:
		fest.phonemes.WriteString("ng ")
	case 144:
		fest.phonemes.WriteString("ng ")
	case 145:
		fest.phonemes.WriteString("l ") // lake, alarm, lapel ??
	case 146:
		fest.phonemes.WriteString("l ") // clock, plus, hello ??
	case 147:
		fest.phonemes.WriteString("w ")
	case 148:
		fest.phonemes.WriteString("r ")
	case 149:
		fest.phonemes.WriteString("iy er ") // clear, hear, year ??

	case 150:
		fest.phonemes.WriteString("er ") // hair, stair, repair ??
	case 151:
		fest.phonemes.WriteString("er ")
	case 152:
		fest.phonemes.WriteString("aa ")
	case 153:
		fest.phonemes.WriteString("ao ")
	case 154:
		fest.phonemes.WriteString("ey ")
	case 155:
		fest.phonemes.WriteString("ay ")
	case 156:
		fest.phonemes.WriteString("oy ")
	case 157:
		fest.phonemes.WriteString("ay ")
	case 158:
		fest.phonemes.WriteString("y ")
	case 159:
		fest.phonemes.WriteString("l ")

	case 160:
		fest.phonemes.WriteString("y uw uw ") // cute, few ??
	case 161:
		fest.phonemes.WriteString("aw ")
	case 162:
		fest.phonemes.WriteString("uw ")
	case 163:
		fest.phonemes.WriteString("aw ")
	case 164:
		fest.phonemes.WriteString("ow ")
	case 165:
		fest.phonemes.WriteString("jh ")
	case 166:
		fest.phonemes.WriteString("v ")
	case 167:
		fest.phonemes.WriteString("z ")
	case 168:
		fest.phonemes.WriteString("zh ")
	case 169:
		fest.phonemes.WriteString("th ")

	case 170:
		fest.phonemes.WriteString("b ")
	case 171:
		fest.phonemes.WriteString("b ")
	case 172:
		fest.phonemes.WriteString("b ")
	case 173:
		fest.phonemes.WriteString("b ")
	case 174:
		fest.phonemes.WriteString("d ")
	case 175:
		fest.phonemes.WriteString("d ")
	case 176:
		fest.phonemes.WriteString("d ")
	case 177:
		fest.phonemes.WriteString("d ")
	case 178:
		fest.phonemes.WriteString("g ")
	case 179:
		fest.phonemes.WriteString("g ")

	case 180:
		fest.phonemes.WriteString("g ")
	case 181:
		fest.phonemes.WriteString("g ")
	case 182:
		fest.phonemes.WriteString("ch ")
	case 183:
		fest.phonemes.WriteString("hh ")
	case 184:
		fest.phonemes.WriteString("hh ")
	case 185:
		fest.phonemes.WriteString("w ") // who, whale, white ??
	case 186:
		fest.phonemes.WriteString("f ")
	case 187:
		fest.phonemes.WriteString("s ")
	case 188:
		fest.phonemes.WriteString("s ")
	case 189:
		fest.phonemes.WriteString("sh ")

	case 190:
		fest.phonemes.WriteString("th ")
	case 191:
		fest.phonemes.WriteString("t ")
	case 192:
		fest.phonemes.WriteString("t ")
	case 193:
		fest.phonemes.WriteString("s ") // partc, costs, robots ??
	case 194:
		fest.phonemes.WriteString("k ")
	case 195:
		fest.phonemes.WriteString("k ")
	case 196:
		fest.phonemes.WriteString("k ")
	case 197:
		fest.phonemes.WriteString("k ")
	case 198:
		fest.phonemes.WriteString("p ")
	case 199:
		fest.phonemes.WriteString("p ")
	}

	fest.stream = append(fest.stream, b)
}

// Flush implements the AtariVoxEngine interface.
func (fest *festival) Flush() {
	if fest.phonemes.Len() == 0 {
		return
	}

	logger.Logf("festival", "stream: %v", fest.stream)
	fest.stream = fest.stream[:0]

	fest.say <- fest.phonemes.String()
	fest.phonemes.Reset()
}
