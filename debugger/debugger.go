package debugger

import (
	"fmt"
	"gopher2600/hardware"
	"gopher2600/hardware/cpu"
	"gopher2600/television"
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

	// commandOnBreak says whether an sequence of commands should run automatically
	// when emulation halts
	commandOnBreak string

	// verbose controls the verbosity of commands that echo machine state
	// TODO: not implemented fully
	verbose bool

	print func(string, ...interface{})
}

// NewDebugger is the preferred method of initialisation for the Debugger structure
func NewDebugger() (*Debugger, error) {
	var err error

	dbg := new(Debugger)

	tv, err := television.NewSDLTV("NTSC", 3)
	if err != nil {
		return nil, err
	}

	dbg.vcs, err = hardware.New(tv)
	if err != nil {
		return nil, err
	}

	dbg.input = make([]byte, 255)
	dbg.breakpoints = newBreakpoints()

	// default verbosity of true -- terse output is for black-belts
	dbg.verbose = true

	dbg.print = func(s string, output ...interface{}) {
		fmt.Printf(s, output...)
	}

	return dbg, nil
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
			// force update of tv image on break
			err = dbg.vcs.TV.ForceUpdate()
			if err != nil {
				return err
			}

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
			_, result, err = dbg.vcs.Step()
			if err != nil {
				return err
			}

			dbg.print("%v\n", result)
		}

		// check for breakpoint. breakpoint check echos the break condition if it
		// matches
		breakpoint = (next && dbg.breakpoints.check(dbg, result))

		// if haltCommand mode and if run state is correct that print haltCommand
		// command(s)
		if dbg.commandOnBreak != "" {
			if (next && !dbg.runUntilBreak) || breakpoint {
				_, _ = dbg.parseInput(dbg.commandOnBreak)
			}
		}

		// expand breakpoint to include step-once/many flag
		breakpoint = breakpoint || !dbg.runUntilBreak
	}

	return nil
}

// parseInput splits the input into individual commands. each command is then
// passed to parseCommand for final processing
func (dbg *Debugger) parseInput(input string) (bool, error) {
	var cont bool
	var err error

	commands := strings.Split(strings.ToUpper(input), ";")
	for i := 0; i < len(commands); i++ {
		cont, err = dbg.parseCommand(commands[i])
		if err != nil {
			return false, err
		}
	}

	return cont, nil
}

// parseCommand scans user input for valid commands and acts upon it. commands
// that cause the emulation to move forward (RUN, STEP) return true for the
// first return value. other commands return false and act upon the command
// immediately. note that the empty string is the same as the STEP command
func (dbg *Debugger) parseCommand(input string) (bool, error) {

	// remove leading/trailing space
	input = strings.TrimSpace(input)

	// if the input is empty then return true, indicating that the emulation
	// should "step" forward once
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

	// normalise variations in syntax
	for i := 0; i < len(parts); i++ {
		// normalise hex notation
		if parts[i][0] == '$' {
			parts[i] = fmt.Sprintf("0x%s", parts[i][1:])
		}
	}

	// most commands do not cause the emulator to step forward
	stepNext := false

	// first entry in parts is the debugging command. switch on this value
	switch parts[0] {
	default:
		return false, fmt.Errorf("%s is not a debugging command", parts[0])

	// control of the debugger

	case "BREAK":
		err := dbg.breakpoints.parseUserInput(dbg, parts)
		if err != nil {
			return false, err
		}

	case "ONBREAK":
		if dbg.commandOnBreak == "" {
			dbg.commandOnBreak = "CPU; TIA; TV"
			dbg.print("auto-command on halt: %s\n", dbg.commandOnBreak)
		} else {
			dbg.commandOnBreak = ""
			dbg.print("no auto-command on halt\n")
		}

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

	case "STEP":
		stepNext = true

	case "TERSE":
		dbg.verbose = false
		dbg.print("verbosity: terse\n")

	case "VERBOSE":
		dbg.verbose = true
		dbg.print("verbosity: verbose\n")

	// information about the machine

	case "CPU":
		dbg.printMachineInfo(dbg.vcs.MC)

	case "TIA":
		dbg.printMachineInfo(dbg.vcs.TIA)

	case "TV":
		dbg.printMachineInfo(dbg.vcs.TV)

	// tv control
	case "SHOW":
		err := dbg.vcs.TV.SetVisibility(true)
		if err != nil {
			return false, err
		}

	case "HIDE":
		err := dbg.vcs.TV.SetVisibility(false)
		if err != nil {
			return false, err
		}
	}

	return stepNext, nil
}
