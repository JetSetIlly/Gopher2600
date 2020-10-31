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

package debugger

import (
	"os"
	"os/signal"
	"strings"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/debugger/script"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/cpu/execution"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/supercharger"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/savekey"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/reflection"
	"github.com/jetsetilly/gopher2600/rewind"
	"github.com/jetsetilly/gopher2600/setup"
	"github.com/jetsetilly/gopher2600/symbols"
)

// Debugger is the basic debugging frontend for the emulation. In order to be
// kind to code that accesses the debugger from a different goroutine (ie. a
// GUI), we try not to reinitialise anything once it has been initialised. For
// example, disassembly on a cartridge change (which can happen at any time)
// updates the Disasm field, it does not reinitialise it.
type Debugger struct {
	VCS    *hardware.VCS
	Disasm *disassembly.Disassembly

	// the bank and formatted result of the last step (cpu or video)
	lastBank   mapper.BankInfo
	lastResult *disassembly.Entry

	// gui, tv and terminal
	tv   *television.Television
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

	// halt conditions
	breakpoints *breakpoints
	traps       *traps
	watches     *watches
	traces      *traces

	// single-fire step traps. these are used for the STEP command, allowing
	// things like "STEP FRAME".
	stepTraps *traps

	// commandOnHalt is the sequence of commands that runs when emulation
	// halts
	commandOnHalt       []*commandline.Tokens
	commandOnHaltStored []*commandline.Tokens

	// commandOnStep is the command to run afer every cpu/video cycle
	commandOnStep       []*commandline.Tokens
	commandOnStepStored []*commandline.Tokens

	// commandOnTrace is the command run whenever a trace condition is met.
	commandOnTrace       []*commandline.Tokens
	commandOnTraceStored []*commandline.Tokens

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
	// because we nest calls to inputLoop on occasion it is better to keep
	// here in the debugger type
	continueEmulation bool

	// halt the emulation immediately. used by HALT command.
	haltImmediately bool

	Rewind *rewind.Rewind
}

// NewDebugger creates and initialises everything required for a new debugging
// session. Use the Start() method to actually begin the session.
func NewDebugger(tv *television.Television, scr gui.GUI, term terminal.Terminal, useSavekey bool) (*Debugger, error) {
	var err error

	dbg := &Debugger{
		tv:   tv,
		scr:  scr,
		term: term,

		// create a minimal lastResult for initialisation
		lastResult: &disassembly.Entry{Result: execution.Result{Final: true}},
	}

	// create a new VCS instance
	dbg.VCS, err = hardware.NewVCS(dbg.tv)
	if err != nil {
		return nil, curated.Errorf("debugger: %v", err)
	}

	// replace player 1 port with savekey
	if useSavekey {
		err = dbg.VCS.RIOT.Ports.AttachPlayer(ports.Player1ID, savekey.NewSaveKey)
		if err != nil {
			return nil, curated.Errorf("debugger: %v", err)
		}
	}

	// create a new disassembly instance
	dbg.Disasm, err = disassembly.NewDisassembly()
	if err != nil {
		return nil, curated.Errorf("debugger: %v", err)
	}

	// set up debugging interface to memory. note that we're reaching deep into
	// another pointer to get the symtable for the memoryDebug instance. this
	// is dangerous if we don't care to reset the symtable when disasm changes.
	// As it is, we only change the disasm poointer in the attachCartridge()
	// function.
	dbg.dbgmem = &memoryDebug{vcs: dbg.VCS, symbols: dbg.Disasm.Symbols}

	// setup reflection monitor
	if b, ok := scr.(reflection.IdentifyReflector); ok {
		dbg.reflect = reflection.NewMonitor(dbg.VCS, b.GetReflectionRenderer())
		dbg.tv.AddReflector(dbg.reflect)
	}

	// plug in rewind system
	dbg.Rewind = rewind.NewRewind(dbg.VCS, dbg)

	// set up breakpoints/traps
	dbg.breakpoints, err = newBreakpoints(dbg)
	if err != nil {
		return nil, curated.Errorf("debugger: %v", err)
	}
	dbg.traps = newTraps(dbg)
	dbg.watches = newWatches(dbg)
	dbg.traces = newTraces(dbg)
	dbg.stepTraps = newTraps(dbg)

	// make synchronisation channels
	//
	// plain debugging terminal causes some missed GUI events. not an issue
	// except the some spurious log messages. it would be nice to get rid
	// of them. extending the length of the channel queue works but that
	// doesn't feel like the correct solution
	//
	// TODO: fix missed GUI events when using plain terminal
	//
	// RawEvents are pushed thick and fast and the channel queue should be
	// pretty lengthy to prevent dropped events (see PushRawEvent() function).
	dbg.events = &terminal.ReadEvents{
		GuiEvents:       make(chan gui.Event, 10),
		GuiEventHandler: dbg.guiEventHandler,
		IntEvents:       make(chan os.Signal, 1),
		RawEvents:       make(chan func(), 1024),
	}

	// connect Interrupt signal to dbg.events.intChan
	signal.Notify(dbg.events.IntEvents, os.Interrupt)

	// connect gui
	err = scr.ReqFeature(gui.ReqSetEventChan, dbg.events.GuiEvents)
	if err != nil {
		if !curated.Is(err, gui.UnsupportedGuiFeature) {
			return nil, curated.Errorf("debugger: %v", err)
		}
	}

	// allocate memory for user input
	dbg.input = make([]byte, 255)

	// add tab completion to terminal
	dbg.term.RegisterTabCompletion(commandline.NewTabCompletion(debuggerCommands))

	// try to add debugger (self) to gui context
	err = dbg.scr.ReqFeature(gui.ReqAddDebugger, dbg)
	if err != nil {
		if !curated.Is(err, gui.UnsupportedGuiFeature) {
			return nil, curated.Errorf("debugger: %v", err)
		}
	}

	return dbg, nil
}

// Start the main debugger sequence.
func (dbg *Debugger) Start(initScript string, cartload cartridgeloader.Loader) error {
	// prepare user interface
	err := dbg.term.Initialise()
	if err != nil {
		return curated.Errorf("debugger: %v", err)
	}
	defer dbg.term.CleanUp()

	err = dbg.attachCartridge(cartload)
	if err != nil {
		return curated.Errorf("debugger: %v", err)
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
				return curated.Errorf("debugger: %v", err)
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
		return curated.Errorf("debugger: %v", err)
	}

	return nil
}

// attachCartridge makes sure that the cartridge loaded into vcs memory and the
// available disassembly/symbols are in sync.
//
// NEVER call vcs.AttachCartridge() or setup.AttachCartridge() except through
// this function
//
// this is the glue that hold the cartridge and disassembly packages together.
// especially important is the repointing of the symbols table in the instance of dbgmem.
func (dbg *Debugger) attachCartridge(cartload cartridgeloader.Loader) error {
	// set OnLoaded function for specific cartridge formats
	cartload.OnLoaded = func(cart mapper.CartMapper) error {
		if _, ok := cart.(*supercharger.Supercharger); ok {
			// !!TODO: it would be nice to see partial disassemblies of supercharger tapes
			// during loading. not completely necessary I don't think, but it would be
			// nice to have.
			err := dbg.Disasm.FromMemory(nil, nil)
			if err != nil {
				return err
			}
			return dbg.tv.Reset()
		} else if pr, ok := cart.(*plusrom.PlusROM); ok {
			if pr.Prefs.NewInstallation {
				fi := gui.PlusROMFirstInstallation{Finish: nil, Cart: pr}
				err := dbg.scr.ReqFeature(gui.ReqPlusROMFirstInstallation, &fi)
				if err != nil {
					if !curated.Is(err, gui.UnsupportedGuiFeature) {
						return curated.Errorf("debugger: %v", err)
					}
				}
			}
		}
		return nil
	}

	err := dbg.scr.ReqFeature(gui.ReqChangingCartridge, true)
	if err != nil {
		if !curated.Is(err, gui.UnsupportedGuiFeature) {
			return curated.Errorf("debugger: %v", err)
		}
	}
	defer func() {
		// we know the gui supports ReqChangingCartridge feature because we've
		// just used it
		_ = dbg.scr.ReqFeature(gui.ReqChangingCartridge, false)
	}()

	err = setup.AttachCartridge(dbg.VCS, cartload)
	if err != nil && !curated.Has(err, cartridge.Ejected) {
		return err
	}

	// attaching a new cartridge always causes the rewind system to reset
	dbg.Rewind.Reset()

	symbols, err := symbols.ReadSymbolsFile(dbg.VCS.Mem.Cart)
	if err != nil {
		logger.Log("symbols", err.Error())
	}

	err = dbg.Disasm.FromMemory(dbg.VCS.Mem.Cart, symbols)
	if err != nil {
		return err
	}

	// repoint debug memory's symbol table
	dbg.dbgmem.symbols = dbg.Disasm.Symbols

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
// next step.
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
