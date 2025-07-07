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
	"os"
	"os/exec"
	"strings"

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/logger"
)

type speakJetAction int

const (
	noAction speakJetAction = iota
	unsupported
	speed
	pitch
	volume
)

type phonemes struct {
	env *environment.Environment
	strings.Builder
	ct int
}

func (p *phonemes) Reset() {
	p.ct = 0
	p.Builder.Reset()
}

func (p *phonemes) WriteString(s string) (int, error) {
	p.ct++
	logger.Logf(p.env, "festival", "phoneme: %s", s)
	return p.Builder.WriteString(s + " ")
}

type festival struct {
	env *environment.Environment

	stdin  io.WriteCloser
	stdout io.ReadCloser

	// echo raw festival commands
	echo io.Writer

	stream   []uint8
	phonemes phonemes

	quit chan bool
	say  chan string
	cmd  chan string

	nextSpeakJetByte speakJetAction

	speed  uint8
	pitch  uint8
	volume uint8
}

const echoToStdErr = false

// nilWriter is an empty writer.
type nilWriter struct{}

func (*nilWriter) Write(p []byte) (n int, err error) {
	return 0, nil
}

// NewFestival creats a new festival instance and starts a new festival process
// which we'll communicate with via a stdin pipe.
func NewFestival(env *environment.Environment) (AtariVoxEngine, error) {
	fest := &festival{
		env: env,
		phonemes: phonemes{
			env: env,
		},
		quit: make(chan bool, 1),
		say:  make(chan string, 1),
		cmd:  make(chan string, 1),
		echo: &nilWriter{},
	}

	if echoToStdErr {
		fest.echo = os.Stderr
	}

	executablePath := env.Prefs.AtariVox.FestivalBinary.Get().(string)
	cmd := exec.Command(executablePath)

	var err error

	fest.stdin, err = cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("festival: %s", err.Error())
	}

	fest.stdout, err = cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("festival: %s", err.Error())
	}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("festival: %s", err.Error())
	}

	go func() {
		for {
			select {
			case <-fest.quit:
				err = cmd.Process.Kill()
				if err != nil {
					logger.Log(fest.env, "festival", err)
				}
				_ = cmd.Wait()
				return

			case text := <-fest.say:
				logger.Logf(fest.env, "festival", "say: %s", text)
				sayphones := fmt.Sprintf("(SayPhones '(%s))", text)
				fest.stdin.Write([]byte(sayphones))
				fest.echo.Write([]byte(sayphones))
				fest.echo.Write([]byte("\n"))

			case command := <-fest.cmd:
				// https://www.cstr.ed.ac.uk/projects/festival/manual/festival_34.html#SEC141
				fest.stdin.Write([]byte(command))
				fest.echo.Write([]byte(command))
				fest.echo.Write([]byte("\n"))
			}
		}
	}()

	fest.reset()

	return fest, nil
}

func (fest *festival) reset() {
	const (
		defaultSpeed  = 114
		defaultPitch  = 88
		defaultVolume = 128
	)

	// no need to do anything if speed, pitch and duration are already at the default values
	if fest.speed != defaultSpeed || fest.pitch != defaultPitch || fest.volume != defaultVolume {
		fest.Flush()
		fest.speed = defaultSpeed
		fest.pitch = defaultPitch
		fest.volume = defaultVolume
		fest.cmd <- fmt.Sprintf("(set! FP_duration %d)", fest.speed)
		fest.cmd <- fmt.Sprintf("(set! FP_F0 %d)", fest.pitch)
	}

	fest.nextSpeakJetByte = noAction
}

// Quit implements the AtariVoxEngine interface.
func (fest *festival) Quit() {
	select {
	case fest.quit <- true:
	default:
	}
}

// SpeakJet implements the AtariVoxEngine interface.
func (fest *festival) SpeakJet(b uint8) {
	// http://festvox.org/festvox-1.2/festvox_18.html

	// https://people.ece.cornell.edu/land/courses/ece4760/Speech/speakjetusermanual.pdf

	fest.stream = append(fest.stream, b)

	switch fest.nextSpeakJetByte {
	case noAction:
	case unsupported:
		fest.nextSpeakJetByte = noAction
		return
	case speed:
		fest.Flush()
		fest.speed = b
		logger.Logf(fest.env, "festival", "speed: %#02x", fest.speed)
		fest.cmd <- fmt.Sprintf("(set! FP_duration %d)", fest.speed)
		fest.nextSpeakJetByte = noAction
		return
	case pitch:
		fest.Flush()
		fest.pitch = b
		logger.Logf(fest.env, "festival", "pitch: %#02x", fest.pitch)
		fest.cmd <- fmt.Sprintf("(set! FP_F0 %d)", fest.pitch)
		fest.nextSpeakJetByte = noAction
		return
	case volume:
		fest.Flush()
		fest.volume = b
		logger.Logf(fest.env, "festival", "volume: %#02x", fest.pitch)
		// volume not implemented directly by a festival command. see Flush()
		// function for how we deal with it as a special case
		fest.nextSpeakJetByte = noAction
	}

	if b >= 215 && b <= 254 {
		logger.Logf(fest.env, "festival", "sound effect: %#02x", b)
		return
	}

	switch b {
	default:
		logger.Logf(fest.env, "festival", "unsupported byte: %#02x", b)

	case 31: // Reset
		logger.Log(fest.env, "festival", "reset")
		fest.reset()

	case 0: // pause 0ms
	case 1: // pause 100ms
		logger.Log(fest.env, "festival", "pause: 100ms")
	case 2: // pause 200ms
		logger.Log(fest.env, "festival", "pause: 200ms")
	case 3: // pause 700ms
		logger.Log(fest.env, "festival", "pause: 700ms")
	case 4: // pause 30ms
		logger.Log(fest.env, "festival", "pause: 30ms")
	case 5: // pause 60ms
		logger.Log(fest.env, "festival", "pause: 60ms")
	case 6: // pause 90ms
		logger.Log(fest.env, "festival", "pause: 90ms")

	// implementing fast/slow and stress/relax is tricky with the festival
	// SayPhones() function. single phonemes do not render very well and so
	// it's best to do without these per-phoneme instructions
	case 7: // Fast
		logger.Log(fest.env, "festival", "fast: not implemented")
	case 8: // Slow
		logger.Log(fest.env, "festival", "slow: not implemented")
	case 14: // Stress
		logger.Log(fest.env, "festival", "stress: not implemented")
	case 15: // Relax
		logger.Log(fest.env, "festival", "relax: not implemented")

	case 20: // volume
		fest.nextSpeakJetByte = volume
	case 21: // speed
		fest.nextSpeakJetByte = speed
	case 22: // pitch
		fest.nextSpeakJetByte = pitch

	case 23: // bend
		logger.Log(fest.env, "festival", "bend: not implemented")
		fest.nextSpeakJetByte = unsupported
	case 24: // PortCtr
		logger.Log(fest.env, "festival", "port ctr: not implemented")
		fest.nextSpeakJetByte = unsupported
	case 25: // Port
		logger.Log(fest.env, "festival", "port: not implemented")
		fest.nextSpeakJetByte = unsupported
	case 26: // Repeat
		logger.Log(fest.env, "festival", "repeat: not implemented")
		fest.nextSpeakJetByte = unsupported
	case 28: // Call Phrase
		logger.Log(fest.env, "festival", "call phrase: not implemented")
		fest.nextSpeakJetByte = unsupported
	case 29: // Goto Phrase
		logger.Log(fest.env, "festival", "goto phrase: not implemented")
		fest.nextSpeakJetByte = unsupported
	case 30: // Delay
		logger.Log(fest.env, "festival", "delay: not implemented")
		fest.nextSpeakJetByte = unsupported

	case 128:
		fest.phonemes.WriteString("iy")
	case 129:
		fest.phonemes.WriteString("ih")

	case 130:
		fest.phonemes.WriteString("ey")
	case 131:
		fest.phonemes.WriteString("eh")
	case 132:
		fest.phonemes.WriteString("ae")
	case 133:
		fest.phonemes.WriteString("en") // cotten ??
	case 134:
		fest.phonemes.WriteString("uh")
	case 135:
		fest.phonemes.WriteString("ah") // hot, clock, fox ??
	case 136:
		fest.phonemes.WriteString("aa")
	case 137:
		fest.phonemes.WriteString("ow")
	case 138:
		fest.phonemes.WriteString("uh") // book, could, should ??  'ah' possibly)
	case 139:
		fest.phonemes.WriteString("uw")

	case 140:
		fest.phonemes.WriteString("m")
	case 141:
		fest.phonemes.WriteString("n")
	case 142:
		fest.phonemes.WriteString("n")
	case 143:
		fest.phonemes.WriteString("ng")
	case 144:
		fest.phonemes.WriteString("ng")
	case 145:
		fest.phonemes.WriteString("l") // lake, alarm, lapel ??
	case 146:
		fest.phonemes.WriteString("l") // clock, plus, hello ??
	case 147:
		fest.phonemes.WriteString("w")
	case 148:
		fest.phonemes.WriteString("r")
	case 149:
		fest.phonemes.WriteString("iy") // clear, hear, year ??
		fest.phonemes.WriteString("er")

	case 150:
		fest.phonemes.WriteString("er") // hair, stair, repair ??
	case 151:
		fest.phonemes.WriteString("er")
	case 152:
		fest.phonemes.WriteString("aa")
	case 153:
		fest.phonemes.WriteString("ao")
	case 154:
		fest.phonemes.WriteString("ey")
	case 155:
		fest.phonemes.WriteString("ay")
	case 156:
		fest.phonemes.WriteString("oy")
	case 157:
		fest.phonemes.WriteString("ay")
	case 158:
		fest.phonemes.WriteString("y")
	case 159:
		fest.phonemes.WriteString("l")

	case 160:
		fest.phonemes.WriteString("y") // cute, few ??
		fest.phonemes.WriteString("uw")
	case 161:
		fest.phonemes.WriteString("aw")
	case 162:
		fest.phonemes.WriteString("uw")
	case 163:
		fest.phonemes.WriteString("aw")
	case 164:
		fest.phonemes.WriteString("ow")
	case 165:
		fest.phonemes.WriteString("jh")
	case 166:
		fest.phonemes.WriteString("v")
	case 167:
		fest.phonemes.WriteString("z")
	case 168:
		fest.phonemes.WriteString("zh")
	case 169:
		fest.phonemes.WriteString("th")

	case 170:
		fest.phonemes.WriteString("b")
	case 171:
		fest.phonemes.WriteString("b")
	case 172:
		fest.phonemes.WriteString("b")
	case 173:
		fest.phonemes.WriteString("b")
	case 174:
		fest.phonemes.WriteString("d")
	case 175:
		fest.phonemes.WriteString("d")
	case 176:
		fest.phonemes.WriteString("d")
	case 177:
		fest.phonemes.WriteString("d")
	case 178:
		fest.phonemes.WriteString("g")
		fest.phonemes.WriteString("g")
	case 179:
		fest.phonemes.WriteString("g")
		fest.phonemes.WriteString("g")

	case 180:
		fest.phonemes.WriteString("g")
		fest.phonemes.WriteString("g")
	case 181:
		fest.phonemes.WriteString("g")
		fest.phonemes.WriteString("g")
	case 182:
		fest.phonemes.WriteString("ch")
	case 183:
		fest.phonemes.WriteString("hh")
	case 184:
		fest.phonemes.WriteString("hh")
	case 185:
		fest.phonemes.WriteString("w") // who, whale, white ??
	case 186:
		fest.phonemes.WriteString("f")
	case 187:
		fest.phonemes.WriteString("s")
	case 188:
		fest.phonemes.WriteString("s")
	case 189:
		fest.phonemes.WriteString("sh")

	case 190:
		fest.phonemes.WriteString("th")
	case 191:
		fest.phonemes.WriteString("t")
	case 192:
		fest.phonemes.WriteString("t")
	case 193:
		fest.phonemes.WriteString("s") // parts, costs, robots ??
	case 194:
		fest.phonemes.WriteString("k")
	case 195:
		fest.phonemes.WriteString("k")
	case 196:
		fest.phonemes.WriteString("k")
	case 197:
		fest.phonemes.WriteString("k")
	case 198:
		fest.phonemes.WriteString("p")
	case 199:
		fest.phonemes.WriteString("p")
	}
}

// Flush implements the AtariVoxEngine interface.
func (fest *festival) Flush() {
	if fest.phonemes.ct == 0 {
		return
	}

	// festival does not work well with single phones (when using the SayPhones
	// function). in these situations we add a 'hh' phone which is audible but
	// not too distracting
	if fest.phonemes.ct == 1 {
		fest.phonemes.WriteString("hh")
	}

	// only say something if volume is above zero
	if fest.volume > 0 {
		fest.say <- fest.phonemes.String()
	} else {
		fest.say <- "pau"
	}

	// clear phonemes
	fest.phonemes.Reset()
}
