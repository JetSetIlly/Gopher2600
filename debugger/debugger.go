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
//
// *** NOTE: all historical versions of this file, as found in any
// git repository, are also covered by the licence, even when this
// notice is not present ***

package debugger

import (
	"os"
	"os/signal"
	"strings"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/debugger/script"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/errors"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/reflection"
	"github.com/jetsetilly/gopher2600/setup"
	"github.com/jetsetilly/gopher2600/symbols"
	"github.com/jetsetilly/gopher2600/television"
)

const defaultOnHalt = "CPU; TV"
const defaultOnStep = "LAST"
const onEmptyInput = "STEP"

// Debugger is the basic debugging frontend for the emulation
type Debugger struct {
	vcs    *hardware.VCS
	disasm *disassembly.Disassembly

	// the cartridge bank that was active before the last instruction was
	// executed
	lastBank int

	// gui, tv and terminal
	tv   television.Television
	scr  gui.GUI
	term terminal.Terminal

	// interface to the vcs memory with additional debugging functions
	// - access to vcs memory from the debugger (eg. peeking and poking) is
	// most fruitfully performed through this structure
	dbgmem *memoryDebug

	// reflection is used to provideo additional information about the
	// emulation. it is inherently slow so should be deactivated if not
	// required
	reflect *reflection.Monitor

	// frame limiter
	lmtr *limiter

	// halt conditions
	breakpoints *breakpoints
	traps       *traps
	watches     *watches

	// single-fire step traps. these are used for the STEP command, allowing
	// things like "STEP FRAME".
	stepTraps *traps

	// commandOnHalt is the sequence of commands that runs when emulation
	// halts. the string is parsed every time it's required, this is
	// inefficient but it gives us enough flexibility to store multiple
	// commands
	commandOnHalt       []*commandline.Tokens
	commandOnHaltStored []*commandline.Tokens

	// commandOnStep is the command to run afer every cpu/video cycle. unlike
	// commandOnHalt, we store these as Tokens. this gives us a little
	// performance improvement
	commandOnStep       []*commandline.Tokens
	commandOnStepStored []*commandline.Tokens

	// quantum to use when stepping/running
	quantum QuantumMode

	// when reading input from the terminal there are other events
	// that need to be monitored
	events *terminal.ReadEvents

	// record user input to a script file
	scriptScribe script.Scribe

	// \/\/\/ inputLoop \/\/\/

	// buffer for user input
	input []byte

	// any error from previous emulation step
	lastStepError bool

	// we accumulate break, trap and watch messsages until we can service them
	// if the strings are empty then no break/trap/watch event has occurred
	breakMessages string
	trapMessages  string
	watchMessages string

	// whether the debugger is to continue with the debugging loop
	// set to false only when debugger is to finish
	running bool

	// continue emulation until a halt condition is encountered
	runUntilHalt bool

	// continue the emulation. this is seemingly only used in the inputLoop but
	// because we nest calls to inputLoop on occassion it is better to keep
	// here in the debugger type
	continueEmulation bool

	// halt the emulation immediately. used by HALT command.
	haltImmediately bool
}

// NewDebugger creates and initialises everything required for a new debugging
// session. Use the Start() method to actually begin the session.
func NewDebugger(tv television.Television, scr gui.GUI, term terminal.Terminal) (*Debugger, error) {
	var err error

	dbg := &Debugger{
		tv:   tv,
		scr:  scr,
		term: term,
	}

	// create a new VCS instance
	dbg.vcs, err = hardware.NewVCS(dbg.tv)
	if err != nil {
		return nil, errors.New(errors.DebuggerError, err)
	}

	// create instance of disassembly -- the same base structure is used
	// for disassemblies subseuquent to the first one.
	dbg.disasm, err = disassembly.FromMemory(dbg.vcs.Mem.Cart, nil)
	if err != nil {
		return nil, errors.New(errors.DebuggerError, err)
	}

	// set up debugging interface to memory. note that we're reaching deep into
	// another pointer to get the symtable for the memoryDebug instance. this
	// is dangerous if we don't care to reset the symtable when disasm changes.
	// As it is, we only change the disasm poointer in the loadCartridge()
	// function.
	dbg.dbgmem = &memoryDebug{mem: dbg.vcs.Mem, symtable: dbg.disasm.Symtable}

	// set up frame limiter
	dbg.lmtr = newLimiter(tv, func() error {
		_, err := dbg.checkEvents(nil)
		return err
	})

	// set up reflection monitor
	if mpx, ok := dbg.scr.(reflection.Renderer); ok {
		dbg.reflect = reflection.NewMonitor(dbg.vcs, mpx)
	} else {
		mpx := &reflection.StubRenderer{}
		dbg.reflect = reflection.NewMonitor(dbg.vcs, mpx)
	}

	// set up breakpoints/traps
	dbg.breakpoints, err = newBreakpoints(dbg)
	if err != nil {
		return nil, errors.New(errors.DebuggerError, err)
	}
	dbg.traps = newTraps(dbg)
	dbg.watches = newWatches(dbg)
	dbg.stepTraps = newTraps(dbg)

	// make synchronisation channels
	dbg.events = &terminal.ReadEvents{
		GuiEvents:       make(chan gui.Event, 2),
		GuiEventHandler: dbg.guiEventHandler,
		IntEvents:       make(chan os.Signal, 1),
		RawEvents:       make(chan func(), 1024),
	}

	// connect Interrupt signal to dbg.events.intChan
	signal.Notify(dbg.events.IntEvents, os.Interrupt)

	// connect gui
	err = scr.SetFeature(gui.ReqSetEventChan, dbg.events.GuiEvents)
	if err != nil {
		return nil, errors.New(errors.DebuggerError, err)
	}

	// allocate memory for user input
	dbg.input = make([]byte, 255)

	// add tab completion to terminal
	dbg.term.RegisterTabCompletion(commandline.NewTabCompletion(debuggerCommands))

	// try to add to gui context
	dbg.scr.SetFeature(gui.ReqAddVCS, dbg.vcs)

	// try to add debugger (self) to gui context
	dbg.scr.SetFeature(gui.ReqAddDebugger, dbg)

	return dbg, nil
}

// Start the main debugger sequence.
func (dbg *Debugger) Start(initScript string, cartload cartridgeloader.Loader) error {
	// prepare user interface
	err := dbg.term.Initialise()
	if err != nil {
		return errors.New(errors.DebuggerError, err)
	}
	defer dbg.term.CleanUp()

	err = dbg.loadCartridge(cartload)
	if err != nil {
		return errors.New(errors.DebuggerError, err)
	}

	dbg.running = true

	// run initialisation script
	if initScript != "" {
		scr, err := script.RescribeScript(initScript)
		if err == nil {
			dbg.term.Silence(true)

			err = dbg.inputLoop(scr, false)
			if err != nil {
				dbg.term.Silence(false)
				return errors.New(errors.DebuggerError, err)
			}

			dbg.term.Silence(false)
		}
	}

	// end script recording gracefully
	defer func() {
		if dbg.scriptScribe.IsActive() {
			_ = dbg.scriptScribe.EndSession()
		}
	}()

	// prepare and run main input loop. inputLoop will not return until
	// debugging session is to be terminated
	err = dbg.inputLoop(dbg.term, false)
	if err != nil {
		return errors.New(errors.DebuggerError, err)
	}

	return nil
}

// loadCartridge makes sure that the cartridge loaded into vcs memory and the
// available disassembly/symbols are in sync.
//
// NEVER call vcs.AttachCartridge() or setup.AttachCartridge() except through
// this function
//
// this is the glue that hold the cartridge and disassembly packages together.
// especially important is the repointing of symtable in the instance of dbgmem
func (dbg *Debugger) loadCartridge(cartload cartridgeloader.Loader) error {
	err := setup.AttachCartridge(dbg.vcs, cartload)
	if err != nil && !errors.Has(err, errors.CartridgeEjected) {
		return err
	}

	symtable, err := symbols.ReadSymbolsFile(cartload.Filename)
	if err != nil {
		dbg.printLine(terminal.StyleError, "%s", err)
		// continuing because symtable is always valid even if err non-nil
	}

	dbg.disasm, err = disassembly.FromMemory(dbg.vcs.Mem.Cart, symtable)
	if err != nil {
		return err
	}

	dbg.scr.SetFeature(gui.ReqAddDisasm, dbg.disasm)

	// repoint debug memory's symbol table
	dbg.dbgmem.symtable = dbg.disasm.Symtable

	return nil
}

// parseInput splits the input into individual commands. each command is then
// passed to parseCommand for processing
//
// interactive argument should be true if  the input that has just come from
// the user (ie. via an interactive terminal). only interactive input will be
// added to a new script file.
//
// auto argument should be true if command is being run as part of ONHALT or
// ONSTEP
//
// returns a boolean stating whether the emulation should continue with the
// next step
func (dbg *Debugger) parseInput(input string, interactive bool, auto bool) (bool, error) {
	var err error
	var continueEmulation bool

	// ignore comments
	if strings.HasPrefix(input, "#") {
		return false, nil
	}

	// divide input if necessary
	commands := strings.Split(input, ";")

	// loop through commands
	for i := 0; i < len(commands); i++ {
		// parse command
		continueEmulation, err = dbg.parseCommand(commands[i], interactive, !auto)
		if err != nil {
			// we don't want to record bad commands in script
			dbg.scriptScribe.Rollback()
			return false, err
		}

		// !!TODO: if continueEmulation is true but there are more commands to
		// parse, what should we do?
	}

	return continueEmulation, nil
}

// SetFPS requests the number frames per second that the emulation should aim
// for. This overrides the frame rate of the specification. A negative FPS
// value restores the specifcications frame rate.
//
// Note that this is only a request, the emulation may not be able to
// achieve that rate.
//
// *Use this in preference to SetFPS() from the television implementation*
func (dbg *Debugger) SetFPS(fps float32) {
	dbg.lmtr.setFPS(fps)
}

// GetReqFPS returens the requested number of frames per second. The limiter
// type has no GetActualFPS() function. Use the equivalent function from the
// television implementation.
//
// *Use this in preference to SetFPS() from the television implementation*
func (dbg *Debugger) GetReqFPS() float32 {
	return dbg.lmtr.getReqFPS()
}
