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
	"github.com/jetsetilly/gopher2600/hardware/television/frameinfo"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/userinput"
)

type Emulation interface {
	UserInput() chan userinput.Event
	VCS() *hardware.VCS
}

type Input interface {
	PushEvent(ports.InputEvent) error
}

type TV interface {
	AddFrameTrigger(f television.FrameTrigger)
	GetFrameInfo() frameinfo.Current
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

	quit      chan bool
	frameNum  chan int
	lastFrame int
}

const headerID = "gopher2600macro"

const (
	headerLineID = iota
	headerNumLines
)

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

	err := mcr.readFile()
	if err != nil {
		return nil, fmt.Errorf("macro: %w", err)
	}

	// attach TV to macro
	mcr.tv.AddFrameTrigger(mcr)

	return mcr, nil
}

func (mcr *Macro) readFile() error {
	f, err := os.Open(mcr.filename)
	if err != nil {
		return err
	}
	buffer, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}

	// convert file contents to an array of lines
	mcr.instructions = strings.Split(string(buffer), "\n")
	if len(mcr.instructions) < headerNumLines {
		return fmt.Errorf("macro: %s: not a macro file", mcr.filename)
	}
	if mcr.instructions[0] != headerID {
		return fmt.Errorf("macro: %s: not a macro file", mcr.filename)
	}

	// ignore version string for now

	// we no longer need the header
	mcr.instructions = mcr.instructions[headerNumLines:]

	return nil
}

func convertAddress(s string) (uint16, error) {
	// convert hex indicator to one that ParseUint can deal with
	if s[0] == '$' {
		s = fmt.Sprintf("0x%s", s[1:])
	}

	// convert address
	a, err := strconv.ParseUint(s, 0, 16)
	return uint16(a), err
}

func convertValue(s string) (uint8, error) {
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

// Run a macro to completion
func (mcr *Macro) Run() {
	go mcr.run()
}

func (mcr *Macro) run() {
	logger.Logf(logger.Allow, "macro", "running %s", mcr.filename)

	// quit instructs the main script loop to end
	var quit bool

	// whether wait frames should be relative or absolute
	var wait func(w int, onEnd func()) bool
	var waitAbsoluteBase int

	waitAbsolute := func(w int, onEnd func()) bool {
		w += waitAbsoluteBase
		if w < mcr.lastFrame {
			return false
		}
		if w == mcr.lastFrame {
			return true
		}

		var done bool
		for !done {
			select {
			case fn := <-mcr.frameNum:
				mcr.lastFrame = fn
				if fn >= w {
					if onEnd != nil {
						onEnd()
					}
					done = true
				}
			case <-mcr.quit:
				done = true
				quit = true
			}
		}

		return true
	}

	waitRelative := func(w int, onEnd func()) bool {
		if w < 0 {
			return false
		}
		if w == 0 {
			return true
		}

		w += mcr.lastFrame

		var done bool
		for !done {
			select {
			case fn := <-mcr.frameNum:
				mcr.lastFrame = fn
				if fn >= w {
					if onEnd != nil {
						onEnd()
					}
					done = true
				}
			case <-mcr.quit:
				done = true
				quit = true
			}
		}

		return true
	}

	// wait is relative by default
	wait = waitRelative

	// number of frames to hold joystick before resuming
	var stickHold int
	var panelHold int
	stickHold = 2
	panelHold = 2

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
			return 0, true, fmt.Errorf("variable '%s' does not exist", n)
		}
		return v.(int), true, nil
	}

instructionLoop:
	for ln := 0; ln < len(mcr.instructions) && !quit; ln++ {
		logf := func(msg string, args ...any) {
			logger.Logf(logger.Allow, "macro", "%s: %d: %s", mcr.filename, ln+headerNumLines+1, fmt.Sprintf(msg, args...))
		}

		// convert argument to number
		number := func(s string) (int, bool) {
			n, err := strconv.Atoi(s)
			if err != nil {
				return 0, false
			}
			return n, true
		}

		select {
		case <-mcr.quit:
			break instructionLoop
		default:
		}

		// ignore commented lines
		if strings.HasPrefix(strings.TrimSpace(mcr.instructions[ln]), "--") {
			continue
		}

		// split input line into a command and its arguments
		toks := strings.Fields(mcr.instructions[ln])
		if len(toks) == 0 {
			continue // for loop
		}
		cmd := toks[0]
		args := toks[1:]

		switch cmd {
		default:
			logf("unrecognised command: %s", cmd)
			return

		case "END":
			break instructionLoop

		case "DO":
			ct := len(args)
			switch ct {
			case 0:
				logf("not enough arguments for %s", cmd)
				return
			case 2:
				fallthrough
			case 1:
				ct, ok := number(args[0])
				if !ok {
					logf("%s counter is not a number: %s", cmd, args[0])
					return
				}
				lp := loop{
					line:     ln,
					countEnd: ct,
				}
				if len(args) == 2 {
					lp.countName = args[1]
					variables[lp.countName] = lp.count
				}
				loops = append(loops, lp)
			default:
				logf("too many arguments for DO")
				return
			}

		case "LOOP":
			if len(args) > 0 {
				logf("too many arguments for %s", cmd)
				return
			}

			// check for a quit signal but don't wait for it
			select {
			case <-mcr.quit:
				break instructionLoop
			default:
			}

			idx := len(loops) - 1
			if idx == -1 {
				logf("LOOP without a DO")
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

		case "FRAMEINFO":
			if len(args) < 1 {
				logf("FRAMEINFO requires at least one argument")
				return
			}
			var s strings.Builder
			for _, a := range args {
				switch a {
				case "NUMBER":
					fmt.Fprintf(&s, "%d ", mcr.tv.GetFrameInfo().FrameNum)
				case "SCANLINES":
					fmt.Fprintf(&s, "%d ", mcr.tv.GetFrameInfo().TotalScanlines)
				case "VSYNC":
					fmt.Fprintf(&s, "%d ", mcr.tv.GetFrameInfo().VSYNCcount)
				case "STABLE":
					fmt.Fprintf(&s, "%v ", mcr.tv.GetFrameInfo().Stable)
				}
			}
			fmt.Println(s.String())

		case "WAIT":
			switch len(args) {
			case 1:
				n, ok := number(args[0])
				if !ok {
					switch args[0] {
					case "RELATIVE":
						wait = waitRelative
					case "ABSOLUTE":
						wait = waitAbsolute
					case "RESET":
						waitAbsoluteBase = mcr.lastFrame
					default:
						logf("%s argument is not a number: %s", cmd, args[0])
						return
					}
				} else {
					if !wait(n, nil) {
						logf("%s %s argument is invalid", cmd, args[0])
						return
					}
				}
			case 0:
				// default to 60 frames
				waitRelative(60, nil)

			default:
				logf("too many arguments for %s", cmd)
				return
			}

		case "SCREENSHOT":
			var s strings.Builder
			for _, c := range args {
				v, ok, err := lookupVariable(c)
				if err != nil {
					logf("%s: %s", cmd, err.Error())
					return
				}
				if ok {
					fmt.Fprintf(&s, "%d", v)
				} else {
					s.WriteString(c)
				}
				s.WriteRune(' ')
			}

			// the filename suffix is all the "words" in string builder
			// joined with an underscore
			filenameSuffix := strings.Join(strings.Fields(s.String()), "_")

			mcr.gui.SetFeature(gui.ReqScreenshot, filenameSuffix)
			waitRelative(10, nil)

		case "QUIT":
			if len(args) > 0 {
				logf("too many arguments for %s", cmd)
				return
			}
			mcr.emulation.UserInput() <- userinput.EventQuit{}

		case "HOLD":
			// sets the number of frames a stick or panel command is held for
			switch len(args) {
			case 2:
				switch args[0] {
				case "STICK":
					var ok bool
					stickHold, ok = number(args[1])
					if !ok {
						logf("%s %s value is not a number: %s", cmd, args[0], args[1])
						return
					}
				case "PANEL":
					var ok bool
					panelHold, ok = number(args[1])
					if !ok {
						logf("%s %s value is not a number: %s", cmd, args[0], args[1])
						return
					}
				default:
					logf("unrecognised option for %s: %s", cmd, args[1])
				}
			default:
				logf("wrong number of arguments for %s", cmd)
				return
			}

		case "FIRE":
			mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Fire, D: true})
			switch len(args) {
			case 1:
				ct, ok := number(args[0])
				if !ok {
					logf("%s counter is not a number: %s", cmd, args[0])
					return
				}
				waitRelative(ct, func() {
					mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Fire, D: false})
				})
			case 0:
				waitRelative(stickHold, nil)
			default:
				logf("too many arguments for %s", cmd)
			}
		case "NOFIRE":
			if len(args) > 0 {
				logf("too many arguments for %s", cmd)
				return
			}
			mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Fire, D: false})
			waitRelative(stickHold, nil)

		case "LEFT":
			if len(args) > 0 {
				logf("too many arguments for %s", cmd)
				return
			}
			mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Left, D: ports.DataStickTrue})
			waitRelative(stickHold, nil)
		case "RIGHT":
			if len(args) > 0 {
				logf("too many arguments for %s", cmd)
				return
			}
			mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Right, D: ports.DataStickTrue})
			waitRelative(stickHold, nil)
		case "UP":
			if len(args) > 0 {
				logf("too many arguments for %s", cmd)
				return
			}
			mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Up, D: ports.DataStickTrue})
			waitRelative(stickHold, nil)
		case "DOWN":
			if len(args) > 0 {
				logf("too many arguments for %s", cmd)
				return
			}
			mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Down, D: ports.DataStickTrue})
			waitRelative(stickHold, nil)
		case "LEFTUP":
			if len(args) > 0 {
				logf("too many arguments for %s", cmd)
				return
			}
			mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.LeftUp, D: ports.DataStickTrue})
			waitRelative(stickHold, nil)
		case "LEFTDOWN":
			if len(args) > 0 {
				logf("too many arguments for %s", cmd)
				return
			}
			mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.LeftDown, D: ports.DataStickTrue})
			waitRelative(stickHold, nil)
		case "RIGHTUP":
			if len(args) > 0 {
				logf("too many arguments for %s", cmd)
				return
			}
			mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.RightUp, D: ports.DataStickTrue})
			waitRelative(stickHold, nil)
		case "RIGHTDOWN":
			if len(args) > 0 {
				logf("too many arguments for %s", cmd)
				return
			}
			mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.RightDown, D: ports.DataStickTrue})
			waitRelative(stickHold, nil)

		case "CENTER":
			fallthrough
		case "CENTRE":
			if len(args) > 0 {
				logf("too many arguments for %s", cmd)
				return
			}
			mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortLeft, Ev: ports.Centre})
			waitRelative(stickHold, nil)

		case "SELECT":
			if len(args) > 0 {
				logf("too many arguments for %s", cmd)
				return
			}
			mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelSelect, D: true})
			waitRelative(panelHold, func() {
				mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelSelect, D: false})
			})

		case "RESET":
			if len(args) > 0 {
				logf("too many arguments for %s", cmd)
				return
			}
			mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelReset, D: true})
			waitRelative(panelHold, func() {
				mcr.input.PushEvent(ports.InputEvent{Port: plugging.PortPanel, Ev: ports.PanelReset, D: false})
			})

		case "POKE":
			if len(args) != 2 {
				logf("not enough arguments for POKE")
				return
			}

			// convert address
			addr, err := convertAddress(args[0])
			if err != nil {
				logf("unrecognised address for POKE: %s", args[0])
				return
			}

			var val uint8

			v, ok, err := lookupVariable(args[1])
			if err != nil {
				logf("%s: %s", cmd, err.Error())
				return
			}
			if ok {
				val = uint8(v)
			} else {
				v, err := convertValue(args[1])
				if err != nil {
					logf("cannot use value for POKE: %s", args[1])
					return
				}
				val = v
			}

			// poke address with value
			mem := mcr.emulation.VCS().Mem
			mem.Poke(addr, val)
		}
	}

	logger.Logf(logger.Allow, "macro", "finished %s", mcr.filename)
}

// Quit forces a running macro (ie. one that has been triggered) to end. Does
// nothing is macro is not currently running
func (mcr *Macro) Quit() {
	select {
	case <-mcr.frameNum:
	default:
	}
	select {
	case mcr.quit <- true:
	default:
	}
}

// Reset restarts the macro
func (mcr *Macro) Reset() error {
	mcr.Quit()
	err := mcr.readFile()
	if err != nil {
		return fmt.Errorf("macro: %w", err)
	}
	mcr.Run()
	return nil
}

// NewFrame implements the television.FrameTrigger interface
func (mcr *Macro) NewFrame(frameInfo frameinfo.Current) error {
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
