package debugger

import (
	"fmt"
	"gopher2600/hardware"
	"gopher2600/hardware/cpu"
	"os"
	"os/signal"
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

	ctrlC := make(chan os.Signal)
	signal.Notify(ctrlC, os.Interrupt)
	go func() {
		for dbg.running {
			<-ctrlC
			if dbg.runUntilBreak == true {
				dbg.runUntilBreak = false
			} else {
				// TODO: interrupt os.Stdin.Read()
				dbg.running = false
			}
		}
	}()

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
		breakpoint = (next && dbg.breakpoints.check(dbg, result)) || !dbg.runUntilBreak
	}

	return nil
}

func (dbg *Debugger) parseInput(input string) (bool, error) {
	// make sure the user has inputted something
	input = strings.TrimSpace(input)
	if input == "" {
		return true, nil
	}

	// divide user input into parts and convert to upper-case for easy parsing
	// input is unchanged in case we need the original user-case
	parts := strings.Split(strings.ToUpper(input), " ")

	// Go's strings.Split() command appends an empty string for every additional
	// space in the input. the for-loop is a little post processing to sanitise
	// the parts array.
	// TODO: perhaps it would be better to write our own Split() function
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
	switch parts[0] {
	default:
		return false, fmt.Errorf("%s is not a debugging command", parts[0])

	case "BREAK":
		err := dbg.breakpoints.parseUserInput(dbg, parts)
		if err != nil {
			return false, err
		}

	case "CPU":
		dbg.print("%v", dbg.vcs.MC)

	case "MEMMAP":
		dbg.print("%v", dbg.vcs.Mem.MemoryMap())

	case "QUIT":
		dbg.running = false

	case "RESET":
		dbg.print("* machine reset\n")
		err := dbg.vcs.Reset()
		if err != nil {
			return false, err
		}

	case "RUN":
		dbg.runUntilBreak = true
		stepNext = true

	}

	return stepNext, nil
}
