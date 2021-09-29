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
	"sync/atomic"
	"time"

	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/debugger/script"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/emulation"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/cpu/execution"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/plusrom"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/supercharger"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/savekey"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/reflection"
	"github.com/jetsetilly/gopher2600/rewind"
	"github.com/jetsetilly/gopher2600/setup"
	"github.com/jetsetilly/gopher2600/userinput"
)

// Debugger is the basic debugging frontend for the emulation. In order to be
// kind to code that accesses the debugger from a different goroutine (ie. a
// GUI), we try not to reinitialise anything once it has been initialised. For
// example, disassembly on a cartridge change (which can happen at any time)
// updates the Disasm field, it does not reinitialise it.
type Debugger struct {
	state atomic.Value // emulation.State

	vcs    *hardware.VCS
	Disasm *disassembly.Disassembly

	// the last loader to be used. we keep a reference to it so we can make
	// sure Close() is called on end
	loader *cartridgeloader.Loader

	// the bank and formatted result of the last step (cpu or video)
	lastBank   mapper.BankInfo
	lastResult *disassembly.Entry

	// gui, tv and terminal
	tv          *television.Television
	scr         gui.GUI
	term        terminal.Terminal
	controllers userinput.Controllers

	// interface to the vcs memory with additional debugging functions
	// - access to vcs memory from the debugger (eg. peeking and poking) is
	// most fruitfully performed through this structure
	dbgmem *memoryDebug

	// reflection is used to provideo additional information about the
	// emulation. it is inherently slow so should be deactivated if not
	// required
	ref *reflection.Gatherer

	// halt conditions
	breakpoints *breakpoints
	traps       *traps
	watches     *watches
	stepTraps   *traps

	// halting is used to coordinate the checking of all halting conditions. it
	// is updated every video cycle as appropriate (ie. not when rewinding)
	halting haltCoordination

	// trace memory access
	traces *traces

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

	// the Rewind system stores and restores machine state.
	Rewind     *rewind.Rewind
	deepPoking chan bool

	// whether the state of the emulation has changed since the last time it
	// was checked - use HasChanged() function
	hasChanged bool

	// \/\/\/ inputLoop \/\/\/

	eventCheckPulse *time.Ticker

	// buffer for user input
	input []byte

	// any error from previous emulation step
	lastStepError bool

	// whether the debugger is to continue with the debugging loop
	// set to false only when debugger is to finish
	//
	// not to be confused with Emulation.Running
	running bool

	// continue emulation until a halt condition is encountered
	//
	// we sometimes think of the halt condition as being paused as in Emulation.Paused
	runUntilHalt bool

	// continue the emulation. this is seemingly only used in the inputLoop()
	// but because we nest calls to inputLoop on occasion it is better to keep
	// here in the debugger type
	continueEmulation bool

	// halt the emulation immediately. used by HALT command.
	//
	// we sometimes think of the halt condition as being paused as in Emulation.Paused
	haltImmediately bool

	// some operations require that the input loop be restarted to make sure
	// continued operation is not inside a video cycle loop
	//
	// we check this frequently inside the inputLoop() function and functions
	// called by inputLoop()
	unwindLoopRestart func() error

	// after a rewinding event it is necessary to make sure the emulation is in
	// the correct place
	catchupContinue func() bool
	catchupEnd      func()
}

// NewDebugger creates and initialises everything required for a new debugging
// session. Use the Start() method to actually begin the session.
func NewDebugger(tv *television.Television, scr gui.GUI, term terminal.Terminal, useSavekey bool) (*Debugger, error) {
	var err error

	dbg := &Debugger{
		tv:   tv,
		scr:  scr,
		term: term,

		// by definition the state of debugger has changed during startup
		hasChanged: true,

		// the ticker to indicate whether we should check for events in the inputLoop
		eventCheckPulse: time.NewTicker(50 * time.Millisecond),
	}

	dbg.state.Store(emulation.Initialising)

	// create a new VCS instance
	dbg.vcs, err = hardware.NewVCS(dbg.tv)
	if err != nil {
		return nil, curated.Errorf("debugger: %v", err)
	}

	// replace player 1 port with savekey
	if useSavekey {
		err = dbg.vcs.RIOT.Ports.Plug(plugging.PortRightPlayer, savekey.NewSaveKey)
		if err != nil {
			return nil, curated.Errorf("debugger: %v", err)
		}
	}

	// set up debugging interface to memory
	dbg.dbgmem = &memoryDebug{
		vcs: dbg.vcs,
	}

	// create a new disassembly instance. also capturing the reference to the
	// disassembly's symbols table
	dbg.Disasm, dbg.dbgmem.sym, err = disassembly.NewDisassembly(dbg.vcs)
	if err != nil {
		return nil, curated.Errorf("debugger: %v", err)
	}

	// create a minimal lastResult for initialisation
	dbg.lastResult = &disassembly.Entry{Result: execution.Result{Final: true}}

	// setup reflection monitor
	dbg.ref = reflection.NewGatherer(dbg.vcs)
	if r, ok := dbg.scr.(reflection.Broker); ok {
		dbg.ref.AddRenderer(r.GetReflectionRenderer())
	}
	dbg.tv.AddFrameTrigger(dbg.ref)

	// plug in rewind system
	dbg.Rewind, err = rewind.NewRewind(dbg.vcs, dbg)
	if err != nil {
		return nil, curated.Errorf("debugger: %v", err)
	}
	dbg.deepPoking = make(chan bool, 1)

	// plug TV BoundaryTrigger into CPU
	dbg.vcs.CPU.AddBoundaryTrigger(dbg.vcs.TV)

	// set up breakpoints/traps
	dbg.breakpoints, err = newBreakpoints(dbg)
	if err != nil {
		return nil, curated.Errorf("debugger: %v", err)
	}
	dbg.traps = newTraps(dbg)
	dbg.watches = newWatches(dbg)
	dbg.traces = newTraces(dbg)
	dbg.stepTraps = newTraps(dbg)

	// make synchronisation channels. RawEvents are pushed thick and fast and
	// the channel queue should be pretty lengthy to prevent dropped events
	// (see PushRawEvent() function).
	dbg.events = &terminal.ReadEvents{
		UserInput:        make(chan userinput.Event, 10),
		UserInputHandler: dbg.userInputHandler,
		IntEvents:        make(chan os.Signal, 1),
		RawEvents:        make(chan func(), 32),
		RawEventsReturn:  make(chan func(), 32),
	}

	// connect Interrupt signal to dbg.events.intChan
	signal.Notify(dbg.events.IntEvents, os.Interrupt)

	// connect gui
	err = dbg.scr.SetFeature(gui.ReqSetEmulation, dbg)
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

	return dbg, nil
}

// VCS implements the emulation.Emulation interface.
func (dbg *Debugger) VCS() emulation.VCS {
	return dbg.vcs
}

// Debugger implements the emulation.Emulation interface.
func (dbg *Debugger) Debugger() emulation.Debugger {
	return dbg
}

// UserInput implements the emulation.Emulation interface.
func (dbg *Debugger) UserInput() chan userinput.Event {
	return dbg.events.UserInput
}

// State implements the emulation.Emulation interface.
func (dbg *Debugger) State() emulation.State {
	return dbg.state.Load().(emulation.State)
}

func (dbg *Debugger) setState(state emulation.State) {
	dbg.tv.SetEmulationState(state)
	dbg.ref.SetEmulationState(state)
	dbg.state.Store(state)
}

// Pause implements the emulation.Emulation interface.
func (dbg *Debugger) Pause(set bool) {
	if set {
		dbg.setState(emulation.Paused)
	} else {
		dbg.setState(emulation.Running)
	}
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

	// end script recording gracefully. this way we don't have to worry too
	// hard about script scribes
	defer func() {
		err := dbg.scriptScribe.EndSession()
		if err != nil {
			logger.Logf("debugger", err.Error())
		}
	}()

	// inputloop will continue until debugger is to be terminated
	for dbg.running {
		err = dbg.inputLoop(dbg.term, false)
		if err != nil {
			return curated.Errorf("debugger: %v", err)
		}

		// handle inputLoopRestart and any on-restart function
		if dbg.unwindLoopRestart != nil {
			err := dbg.unwindLoopRestart()
			if err != nil {
				return curated.Errorf("debugger: %v", err)
			}
			dbg.unwindLoopRestart = nil
		} else {
			dbg.running = false
		}
	}

	// make sure any cartridge loader has been finished with
	if dbg.loader != nil {
		err = dbg.loader.Close()
		if err != nil {
			return curated.Errorf("debugger: %v", err)
		}
	}

	return nil
}

// HasChanged returns true if emulation state has changed since last call to
// the function.
func (dbg *Debugger) HasChanged() bool {
	v := dbg.hasChanged
	dbg.hasChanged = false
	return v
}

// reset of VCS should go through this function to makes sure debugger is reset
// accordingly also. note that debugging features (breakpoints, etc.) are not
// reset.
func (dbg *Debugger) reset() error {
	err := dbg.vcs.Reset()
	if err != nil {
		return err
	}
	dbg.Rewind.Reset()
	dbg.lastResult = &disassembly.Entry{Result: execution.Result{Final: true}}
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
func (dbg *Debugger) attachCartridge(cartload cartridgeloader.Loader) (e error) {
	dbg.setState(emulation.Initialising)
	defer func() {
		if dbg.runUntilHalt && e == nil {
			dbg.setState(emulation.Running)
		} else {
			dbg.setState(emulation.Paused)
		}
	}()

	// close any existing loader before continuing
	if dbg.loader != nil {
		err := dbg.loader.Close()
		if err != nil {
			return err
		}
	}
	dbg.loader = &cartload

	// set VCSHook for specific cartridge formats
	cartload.VCSHook = func(cart mapper.CartMapper, event mapper.Event, args ...interface{}) error {
		if _, ok := cart.(*supercharger.Supercharger); ok {
			switch event {
			case mapper.EventSuperchargerLoadStarted:
				// not required for the debugger
			case mapper.EventSuperchargerFastloadEnded:
				// the supercharger ROM will eventually start execution from the PC
				// address given in the supercharger file

				// CPU execution has been interrupted. update state of CPU
				dbg.vcs.CPU.Interrupted = true

				// the interrupted CPU means it never got a chance to
				// finalise the result. we force that here by simply
				// setting the Final flag to true.
				dbg.vcs.CPU.LastResult.Final = true

				// we've already obtained the disassembled lastResult so we
				// need to change the final flag there too
				dbg.lastResult.Result.Final = true

				// call function to complete tape loading procedure
				callback := args[0].(supercharger.FastLoaded)
				err := callback(dbg.vcs.CPU, dbg.vcs.Mem.RAM, dbg.vcs.RIOT.Timer)
				if err != nil {
					return err
				}

				// (re)disassemble memory on TapeLoaded error signal
				err = dbg.Disasm.FromMemory()
				if err != nil {
					return err
				}
			case mapper.EventSuperchargerSoundloadStarted:
				// not required for the debugger
			case mapper.EventSuperchargerSoundloadEnded:
				// !!TODO: it would be nice to see partial disassemblies of supercharger tapes
				// during loading. not completely necessary I don't think, but it would be
				// nice to have.
				err := dbg.Disasm.FromMemory()
				if err != nil {
					return err
				}
				return dbg.tv.Reset(true)
			case mapper.EventSuperchargerSoundloadRewind:
				// not required for the debugger
			default:
				logger.Logf("debugger", "unhandled hook event for supercharger (%v)", event)
			}
		} else if pr, ok := cart.(*plusrom.PlusROM); ok {
			switch event {
			case mapper.EventPlusROMInserted:
				if pr.Prefs.NewInstallation {
					fi := gui.PlusROMFirstInstallation{Finish: nil, Cart: pr}
					err := dbg.scr.SetFeature(gui.ReqPlusROMFirstInstallation, &fi)
					if err != nil {
						if !curated.Is(err, gui.UnsupportedGuiFeature) {
							return curated.Errorf("debugger: %v", err)
						}
					}
				}
			case mapper.EventPlusROMNetwork:
				// not required for debugger
			default:
				logger.Logf("debugger", "unhandled hook event for plusrom (%v)", event)
			}
		}
		return nil
	}

	// reset of vcs is implied with attach cartridge
	err := setup.AttachCartridge(dbg.vcs, cartload)
	if err != nil && !curated.Has(err, cartridge.Ejected) {
		logger.Log("attach", err.Error())

		// an error has occurred so attach the ejected cartridge
		//
		// !TODO: a special error cartridge to make it more obvious what has happened
		err = setup.AttachCartridge(dbg.vcs, cartridgeloader.Loader{})
		if err != nil {
			return err
		}
	}

	// disassemble newly attached cartridge
	err = dbg.Disasm.FromMemory()
	if err != nil {
		return err
	}

	// make sure everything is reset after disassembly
	dbg.reset()

	return nil
}

func (dbg *Debugger) hotload() (e error) {
	// tell GUI that we're in the initialistion phase
	dbg.setState(emulation.Initialising)
	defer func() {
		if dbg.runUntilHalt && e == nil {
			dbg.setState(emulation.Running)
		} else {
			dbg.setState(emulation.Paused)
		}
	}()

	// close any existing loader before continuing
	if dbg.loader != nil {
		err := dbg.loader.Close()
		if err != nil {
			return err
		}
	}

	cartload, err := cartridgeloader.NewLoader(dbg.vcs.Mem.Cart.Filename, dbg.vcs.Mem.Cart.ID())
	if err != nil {
		return err
	}
	dbg.loader = &cartload

	err = dbg.vcs.Mem.Cart.HotLoad(cartload)
	if err != nil {
		return err
	}

	// disassemble newly attached cartridge
	err = dbg.Disasm.FromMemory()
	if err != nil {
		return err
	}

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
func (dbg *Debugger) parseInput(input string, interactive bool, auto bool) error {
	var err error

	// ignore comments
	if strings.HasPrefix(input, "#") {
		return nil
	}

	// divide input if necessary
	commands := strings.Split(input, ";")

	// loop through commands
	for i := 0; i < len(commands); i++ {
		// parse command
		err = dbg.parseCommand(commands[i], interactive, !auto)
		if err != nil {
			// we don't want to record bad commands in script
			dbg.scriptScribe.Rollback()
			return err
		}
	}

	return nil
}
