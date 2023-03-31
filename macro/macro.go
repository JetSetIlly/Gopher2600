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

package macro

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/userinput"
)

type Emulation interface {
	UserInput() chan userinput.Event
	VCS() *hardware.VCS
}

type Input interface {
	PushEvent(ports.InputEvent) error
	AllowPushedEvents(bool)
}

type TV interface {
	AddFrameTrigger(f television.FrameTrigger)
	GetFrameInfo() television.FrameInfo
}

type GUI interface {
	SetFeature(request gui.FeatureReq, args ...gui.FeatureReqData) error
}

// Macro is a type that allows control of an emulation from a series of instructions
type Macro struct {
	emulation Emulation
	input     Input
	tv        TV
	gui       GUI

	filename     string
	instructions []string

	quit     chan bool
	frameNum chan int
}

const (
	headerLineID = iota
	headerLineVersion
	headerNumLines
)

const headerID = "gopher2600macro"

// NewMacro is the preferred method of initialisation for the Macro type
func NewMacro(filename string, emulation Emulation, input Input, tv TV, gui GUI) (*Macro, error) {
	mcr := &Macro{
		emulation: emulation,
		input:     input,
		tv:        tv,
		gui:       gui,
		filename:  filename,
		quit:      make(chan bool),
		frameNum:  make(chan int),
	}

	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("macro: %w", err)
	}
	buffer, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("macro: %w", err)
	}
	err = f.Close()
	if err != nil {
		return nil, fmt.Errorf("macro: %w", err)
	}

	// convert file contents to an array of lines
	mcr.instructions = strings.Split(string(buffer), "\n")
	if len(mcr.instructions) < headerNumLines {
		return nil, fmt.Errorf("macro: %s: not a macro file", filename)
	}
	if mcr.instructions[0] != headerID {
		return nil, fmt.Errorf("macro: %s: not a macro file", filename)
	}

	// ignore version string for now

	// we no longer need the header
	mcr.instructions = mcr.instructions[headerNumLines:]

	// allow pushed events to the VCS input system
	mcr.input.AllowPushedEvents(true)
	mcr.tv.AddFrameTrigger(mcr)

	return mcr, nil
}

// Run a macro to completion
func (mcr *Macro) Run() {
	log := func(ln int, msg string) {
		logger.Logf("macro", "%s: %d: %s", mcr.filename, ln+headerNumLines, msg)
	}

	// wait function is used by the WAIT macro instruction but is also used by
	// the controller instructions (LEFT, FIRE, etc.) to indicate a short wait
	// of two frames before moving onto the next instruction in the macro
	// script. this ensures that the controller input has the chance to take
	// effect in the emulation
	wait := func(w int) bool {
		target := w + <-mcr.frameNum

		var done bool
		for !done {
			select {
			case fn := <-mcr.frameNum:
				if fn >= target {
					done = true
				}
			case <-mcr.quit:
				return true
			}
		}
		return false
	}

	convertAddress := func(s string) (uint16, error) {
		// convert hex indicator to one that ParseUint can deal with
		if s[0] == '$' {
			s = fmt.Sprintf("0x%s", s[1:])
		}

		// convert address
		a, err := strconv.ParseUint(s, 0, 16)
		return uint16(a), err
	}

	convertValue := func(s string) (uint8, error) {
		// convert hex indicator to one that ParseUint can deal with
		if s[0] == '$' {
			s = fmt.Sprintf("0x%s", s[1:])
		}

		// convert address
		a, err := strconv.ParseUint(s, 0, 8)
		return uint8(a), err
	}

	type loop struct {
		line int

		// loop counters count upwards because it is more natural when
		// referencing the counter value to think of the counter as counting
		// upwards
		count    int
		countEnd int

		// if loop counter has been named then we need to know it so that we can
		// update the entry in the variables table
		countName string
	}

	go func() {
		var loops []loop
		variables := make(map[string]any)

		lookupVariable := func(n string) (int, bool, error) {
			// convert value
			if n[0] != '%' {
				return 0, false, nil
			}

			n = n[1:]
			v, ok := variables[n]
			if !ok {
				return 0, true, fmt.Errorf("cannot use variable '%s' in POKE because it does not exist", n)
			}
			return v.(int), true, nil
		}

		for ln := 0; ln < len(mcr.instructions); ln++ {
			s := mcr.instructions[ln]

			toks := strings.Fields(s)
			if len(toks) == 0 {
				continue // for loop
			}

			switch toks[0] {
			default:
				log(ln, fmt.Sprintf("unrecognised command: %s", toks[0]))
				return

			case "--":
				// ignore comment lines

			case "DO":
				tl := len(toks)
				switch tl {
				case 1:
					log(ln, "too few arguments for DO")
					return
				case 3:
					fallthrough
				case 2:
					ct, err := strconv.Atoi(toks[1])
					if err != nil {
						log(ln, err.Error())
						return
					}
					lp := loop{
						line:     ln,
						countEnd: ct,
					}
					if tl == 3 {
						lp.countName = toks[2]
						variables[lp.countName] = lp.count
					}
					loops = append(loops, lp)
				default:
					log(ln, "too many arguments for DO")
					return
				}

			case "LOOP":
				if len(toks) > 1 {
					log(ln, "too many arguments for LOOP")
					return
				}

				// check for a quit signal but don't wait for it
				select {
				case <-mcr.quit:
					return
				default:
				}

				idx := len(loops) - 1
				if idx == -1 {
					log(ln, "LOOP without a DO")
					return
				}

				lp := &loops[idx]
				lp.count++

				if lp.count < lp.countEnd {
					// loop is ongoing so return to start of loop
					ln = lp.line

					// update named variable
					if lp.countName != "" {
						variables[lp.countName] = lp.count
					}
				} else {
					// loop has ended. remove from loop stack and delete variable name
					loops = loops[:idx]
					delete(variables, lp.countName)
				}

			case "WAIT":
				// default to 60 frames
				w := 60

				switch len(toks) {
				case 2:
					var err error
					w, err = strconv.Atoi(toks[1])
					if err != nil {
						log(ln, err.Error())
						return
					}
					fallthrough

				case 1:
					if wait(w) {
						return
					}

				default:
					log(ln, "too many arguments for WAIT")
					return
				}

			case "SCREENSHOT":
				var s strings.Builder
				for _, c := range toks[1:] {
					v, ok, err := lookupVariable(c)
					if err != nil {
						log(ln, err.Error())
						return
					}
					if ok {
						s.WriteString(fmt.Sprintf("%d", v))
					} else {
						s.WriteString(c)
					}
					s.WriteRune(' ')
				}

				// the filename suffix is all the "words" in string builder
				// joined with an underscore
				filenameSuffix := strings.Join(strings.Fields(s.String()), "_")

				mcr.gui.SetFeature(gui.ReqScreenshot, filenameSuffix)
				wait(10)

			case "QUIT":
				if len(toks) > 1 {
					log(ln, "too many arguments for QUIT")
					return
				}
				mcr.emulation.UserInput() <- userinput.EventQuit{}

			case "FIRE":
				mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Fire, D: true})
				wait(2)
			case "NOFIRE":
				mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Fire, D: false})
				wait(2)
			case "LEFT":
				mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Left, D: ports.DataStickTrue})
				wait(2)
			case "RIGHT":
				mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Right, D: ports.DataStickTrue})
				wait(2)
			case "UP":
				mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Up, D: ports.DataStickTrue})
				wait(2)
			case "DOWN":
				mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Down, D: ports.DataStickTrue})
				wait(2)
			case "LEFTUP":
				mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.LeftUp, D: ports.DataStickTrue})
				wait(2)
			case "LEFTDOWN":
				mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.LeftDown, D: ports.DataStickTrue})
				wait(2)
			case "RIGHTUP":
				mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.RightUp, D: ports.DataStickTrue})
				wait(2)
			case "RIGHTDOWN":
				mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.RightDown, D: ports.DataStickTrue})
				wait(2)
			case "CENTER":
				fallthrough
			case "CENTRE":
				mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Centre})
				wait(2)
			case "SELECT":
				held := false
				if len(toks) == 1 {
					held = true
				}
				mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelSelect, D: held})
				wait(2)
			case "RESET":
				held := false
				if len(toks) == 1 {
					held = true
				}
				mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelReset, D: held})
				wait(2)
			case "POKE":
				if len(toks) != 3 {
					log(ln, "not enough arguments for POKE")
					return
				}

				// convert address
				addr, err := convertAddress(toks[1])
				if err != nil {
					log(ln, fmt.Sprintf("unrecognised address for POKE: %s", toks[1]))
					return
				}

				var val uint8

				v, ok, err := lookupVariable(toks[2])
				if err != nil {
					log(ln, err.Error())
					return
				}
				if ok {
					val = uint8(v)
				} else {
					v, err := convertValue(toks[2])
					if err != nil {
						log(ln, fmt.Sprintf("cannot use value for POKE: %s", toks[2]))
						return
					}
					val = uint8(v)
				}

				// poke address with value
				mem := mcr.emulation.VCS().Mem
				mem.Poke(addr, val)
			}
		}
	}()
}

// Quit forces a running macro (ie. one that has been triggered) to end. Does
// nothing is macro is not currently running
func (mcr *Macro) Quit() {
	select {
	case mcr.quit <- true:
	default:
	}
}

// NewFrame implements the television.FrameTrigger interface
func (mcr *Macro) NewFrame(frameInfo television.FrameInfo) error {
	// drain any frameNum channel before pushing a new value
	select {
	case <-mcr.frameNum:
	default:
	}
	select {
	case mcr.frameNum <- frameInfo.FrameNum:
	default:
	}
	return nil
}
