package debugger

import (
	"fmt"
	"headlessVCS/hardware"
	"headlessVCS/hardware/cpu"
	"os"
	"strings"
)

// Debugger is the basic debugging frontend for the emulation
type Debugger struct {
	vcs     *hardware.VCS
	running bool
	input   []byte

	breakpoints   *breakpoints
	runUntilBreak bool

	print func(string, ...interface{})
}

// NewDebugger is the preferred method of initialisation for the Debugger structure
func NewDebugger() *Debugger {
	dbg := new(Debugger)
	dbg.vcs = hardware.NewVCS()
	dbg.input = make([]byte, 255)
	dbg.breakpoints = newBreakpoints()

	dbg.print = func(s string, output ...interface{}) {
		fmt.Printf(s, output...)
	}

	return dbg
}

// Start the main debugger sequence
func (dbg *Debugger) Start(filename string) error {
	err := dbg.vcs.AttachCartridge(filename)
	if err != nil {
		return err
	}

	err = dbg.inputLoop()
	if err != nil {
		return err
	}

	return nil
}

func (dbg *Debugger) inputLoop() error {
	var err error
	var result *cpu.InstructionResult
	breakpoint := true
	next := true

	dbg.running = true
	for dbg.running {
		if breakpoint {
			// reset run until break condition
			dbg.runUntilBreak = false

			// get user input
			dbg.print("[0x%04x] > ", dbg.vcs.MC.PC.ToUint16())
			n, err := os.Stdin.Read(dbg.input)
			if err != nil {
				return err
			}

			// parse user input
			next, err = dbg.parseInput(string(dbg.input[:n-1]))
			if err != nil {
				dbg.print("* %s\n", err)
			}

			// prepare for next loop
			breakpoint = false
		} else {
			// TODO: check for user interrupt
		}

		// move emulation on one step
		if next {
			result, err = dbg.vcs.Step()
			if err != nil {
				return err
			}
			dbg.print("%v\n", result)
		}

		// check for breakpoint
		breakpoint = !dbg.runUntilBreak || dbg.breakpoints.check(dbg.vcs, result)
	}

	return nil
}

func (dbg *Debugger) parseInput(input string) (bool, error) {
	// make sure the user has inputted something
	input = strings.TrimSpace(input)
	if input == "" {
		return true, nil
	}

	// divide user input into parts -- Go's strings.Split() command appends an empty
	// string for every additional space in the input. the for-loop is a little
	// post processing to sanitise the parts array.
	// TODO: maybe better to write our own Split() function
	parts := strings.Split(input, " ")
	partsb := make([]string, 0)
	for i := 0; i < len(parts); i++ {
		if parts[i] != "" {
			partsb = append(partsb, parts[i])
		}
	}
	parts = partsb

	// most commands do not cause the emulator to step forward
	stepNext := false

	// first entry in parts is the debugging command. switch on this value
	switch strings.ToUpper(parts[0]) {
	default:
		return false, fmt.Errorf("%s is not a debugging command", strings.ToUpper(parts[0]))

	case "BREAK":
		err := dbg.breakpoints.add(parts)
		if err != nil {
			return false, err
		}

	case "CPU":
		dbg.print("%v\n", dbg.vcs.MC)

	case "QUIT":
		dbg.running = false

	case "RESET":
		dbg.print("* machine reset\n")
		dbg.vcs.Reset()

	case "RUN":
		dbg.runUntilBreak = true
		stepNext = true

	}

	return stepNext, nil
}
