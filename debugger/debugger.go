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
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jetsetilly/gopher2600/bots/wrangler"
	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/comparison"
	coprocDev "github.com/jetsetilly/gopher2600/coprocessor/developer"
	coprocDisasm "github.com/jetsetilly/gopher2600/coprocessor/disassembly"
	"github.com/jetsetilly/gopher2600/curated"
	"github.com/jetsetilly/gopher2600/debugger/dbgmem"
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
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/patch"
	"github.com/jetsetilly/gopher2600/prefs"
	"github.com/jetsetilly/gopher2600/recorder"
	"github.com/jetsetilly/gopher2600/reflection"
	"github.com/jetsetilly/gopher2600/reflection/counter"
	"github.com/jetsetilly/gopher2600/resources/unique"
	"github.com/jetsetilly/gopher2600/rewind"
	"github.com/jetsetilly/gopher2600/setup"
	"github.com/jetsetilly/gopher2600/tracker"
	"github.com/jetsetilly/gopher2600/userinput"
	"github.com/jetsetilly/gopher2600/wavwriter"
)

// Debugger is the basic debugging frontend for the emulation. In order to be
// kind to code that accesses the debugger from a different goroutine (ie. a
// GUI), we try not to reinitialise anything once it has been initialised. For
// example, disassembly on a cartridge change (which can happen at any time)
// updates the Disasm field, it does not reinitialise it.
type Debugger struct {
	// arguments from the command line
	opts CommandLineOptions

	// current mode of the emulation. use setMode() to set the value
	mode atomic.Value // emulation.Mode

	// when playmode is entered without a ROM specified we send the GUI a
	// ReqROMSelector request. we create the forcedROMselection channel and
	// wait for a response from the InsertCartridge() function. sending and
	// receiving on this channel occur in the same goroutine so the channel
	// must be buffered
	forcedROMselection chan bool

	// state is an atomic value because we need to be able to read it from the
	// GUI thread (see State() function)
	state atomic.Value // emulation.State

	// preferences for the emulation
	Prefs *Preferences

	// reference to emulated hardware. this pointer never changes through the
	// life of the emulation even though the hardware may change and the
	// components may change (during rewind for example)
	vcs *hardware.VCS

	// the last loader to be used. we keep a reference to it so we can make
	// sure Close() is called on end
	loader *cartridgeloader.Loader

	// supercharger multiload byte to force on mapper.EventSuperchargerLoadStarted
	//
	// a value of -1 means a value will not be forced.
	multiload int

	// gameplay recorder/playback
	recorder *recorder.Recorder
	playback *recorder.Playback

	// comparison emulator
	comparison *comparison.Comparison

	// GUI, terminal and controllers
	gui         gui.GUI
	term        terminal.Terminal
	controllers *userinput.Controllers

	// bots coordinator
	bots *wrangler.Bots

	// when reading input from the terminal there are other events
	// that need to be monitored
	events *terminal.ReadEvents

	// how often the events field should be checked
	eventCheckPulse *time.Ticker

	// cartridge disassembly
	//
	// * allocated when entering debugger mode
	Disasm       *disassembly.Disassembly
	CoProcDisasm *coprocDisasm.Disassembly
	CoProcDev    *coprocDev.Developer

	// the live disassembly entry. updated every CPU step or on halt (which may
	// be mid instruction)
	//
	// we use this so that we can display the instruction as it exists in the
	// CPU at any given time, which means we can see the partially decoded
	// instruction and easily keep track of how many cycles an instruction has
	// taken so far.
	//
	// for CPU quantum the liveDisasmEntry will be the full entry that is in
	// the disassembly
	//
	// for both quantums we update liveDisasmEntry with the ExecutedEntry()
	// function in the disassembly package
	liveDisasmEntry *disassembly.Entry

	// the live bank information. updated every CPU step or on halt (which may
	// be mid instruction)
	liveBankInfo mapper.BankInfo

	// the television coords of the last CPU instruction
	cpuBoundaryLastInstruction coords.TelevisionCoords

	// interface to the vcs memory with additional debugging functions
	// - access to vcs memory from the debugger (eg. peeking and poking) is
	// most fruitfully performed through this structure
	dbgmem *dbgmem.DbgMem

	// reflection is used to provideo additional information about the
	// emulation. it is inherently slow so should be deactivated if not
	// required
	//
	// * allocated when entering debugger mode
	ref *reflection.Reflector

	// closely related to the relection system is the counter. generally
	// updated, cleared, etc. at the same time as the reflection system.
	counter *counter.Counter

	// halting is used to coordinate the checking of all halting conditions. it
	// is updated every video cycle as appropriate (ie. not when rewinding)
	halting *haltCoordination

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

	// stepQuantum to use when stepping/running
	stepQuantum Quantum

	// catchupQuantum differs from the quantum field in that it only applies in
	// the catchupLoop (part of the rewind system). it is set just before the
	// rewind process is started.
	//
	// the value it is set to depends on the context. For the STEP BACK command
	// it is set to the current stepQuantum
	//
	// for PushGoto() the quantum is set to QuantumVideo, while for
	// PushRewind() it is set to the current stepQuantum
	catchupQuantum Quantum

	// record user input to a script file
	scriptScribe script.Scribe

	// the Rewind system stores and restores machine state
	Rewind *rewind.Rewind

	// the amount we rewind by is dependent on how fast the mouse wheel is
	// moving or for how long the keyboard (or gamepad bumpers) have been
	// depressed.
	//
	// when rewinding by mousewheel, events are likely to be sent during the
	// rewind catchup loop so we accumulate the mousewheel delta and rewind
	// when we return to the normal loop
	//
	// keyboard/bumper rewind is slightly different. for every machine cycle
	// (in the normal playloop - not the catchup loop) that the keyboard is
	// held down we increase (or decrease when going backwards) the
	// accumulation value. we use this to determine how quickly the rewind
	// should progress. the accumulation value is zeroed when the key/bumpers
	// are released
	//
	// * playmode only
	rewindMouseWheelAccumulation int
	rewindKeyboardAccumulation   int

	// audio tracker stores audio state over time
	Tracker *tracker.Tracker

	// whether the state of the emulation has changed since the last time it
	// was checked - use HasChanged() function
	hasChanged bool

	// \/\/\/ debugger inputLoop \/\/\/

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
	continueEmulation bool

	// halt the emulation immediately. used by HALT command.
	//
	// we sometimes think of the halt condition as being paused as in Emulation.Paused
	haltImmediately bool

	// in very specific circumstances it is necessary to step out of debugger
	// loop if it's in the middle of a video step. this happens very rarely but
	// is necessary in order to *feel* natural to the user - without it it can
	// sometimes require an extra STEP instruction to continue, which can be
	// confusing
	//
	// it can be thought of as a lightweight unwind loop function
	//
	// it is currently used only to implement stepping (in instruction quantum)
	// when the emulation state is "inside" the WSYNC
	stepOutOfVideoStepInputLoop bool

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

	// the debugger catchup loop will end on a video cycle if necessary. this
	// is what we want in most situations but occasionally it is useful to stop
	// on an instruction boundary. catchupEndAdj will ensure that the debugger
	// halts on an instruction boundary
	//
	// the value will reset to false at the end of a catchup loop
	catchupEndAdj bool
}

// CreateUserInterface is used to initialise the user interface used by the emulation.
type CreateUserInterface func(emulation.Emulation) (gui.GUI, terminal.Terminal, error)

// NewDebugger creates and initialises everything required for a new debugging
// session.
//
// It should be followed up with a call to AddUserInterface() and call the
// Start() method to actually begin the emulation.
func NewDebugger(create CreateUserInterface, opts CommandLineOptions) (*Debugger, error) {
	dbg := &Debugger{
		// copy of the arguments
		opts: opts,

		// by definition the state of debugger has changed during startup
		hasChanged: true,

		// the ticker to indicate whether we should check for events in the inputLoop
		eventCheckPulse: time.NewTicker(50 * time.Millisecond),

		// supercharger multiload byte
		multiload: *opts.Multiload,
	}

	// emulator is starting in the "none" mode (the advangatge of this is that
	// we get to set the underlying type of the atomic.Value early before
	// anyone has a change to call State() or Mode() from another thread)
	dbg.state.Store(emulation.EmulatorStart)
	dbg.mode.Store(emulation.ModeNone)

	var err error

	// load preferences
	dbg.Prefs, err = newPreferences()
	if err != nil {
		return nil, curated.Errorf("debugger: %v", err)
	}

	// creat a new television. this will be used during the initialisation of
	// the VCS and not referred to directly again
	tv, err := television.NewTelevision(*opts.Spec)
	if err != nil {
		return nil, curated.Errorf("debugger: %v", err)
	}

	// create a new VCS instance
	dbg.vcs, err = hardware.NewVCS(tv, nil)
	if err != nil {
		return nil, curated.Errorf("debugger: %v", err)
	}

	// create userinput/controllers handler
	dbg.controllers = userinput.NewControllers()
	dbg.controllers.AddInputHandler(dbg.vcs.Input)

	// create bot coordinator
	dbg.bots = wrangler.NewBots(dbg.vcs.Input, dbg.vcs.TV)

	// set up debugging interface to memory
	dbg.dbgmem = &dbgmem.DbgMem{
		VCS: dbg.vcs,
	}

	// create a new disassembly instance
	dbg.Disasm, dbg.dbgmem.Sym, err = disassembly.NewDisassembly(dbg.vcs)
	if err != nil {
		return nil, curated.Errorf("debugger: %v", err)
	}

	// create a minimal lastResult for initialisation
	dbg.liveDisasmEntry = &disassembly.Entry{Result: execution.Result{Final: true}}

	// halting coordination
	dbg.halting, err = newHaltCoordination(dbg)
	if err != nil {
		return nil, curated.Errorf("debugger: %v", err)
	}

	// traces
	dbg.traces = newTraces(dbg)

	// make synchronisation channels. RawEvents are pushed thick and fast and
	// the channel queue should be pretty lengthy to prevent dropped events
	// (see PushRawEvent() function).
	dbg.events = &terminal.ReadEvents{
		UserInput:          make(chan userinput.Event, 10),
		UserInputHandler:   dbg.userInputHandler,
		IntEvents:          make(chan os.Signal, 1),
		RawEvents:          make(chan func(), 1024),
		RawEventsImmediate: make(chan func(), 1024),
	}

	// connect Interrupt signal to dbg.events.intChan
	signal.Notify(dbg.events.IntEvents, os.Interrupt)

	// allocate memory for user input
	dbg.input = make([]byte, 255)

	// create GUI
	dbg.gui, dbg.term, err = create(dbg)
	if err != nil {
		return nil, curated.Errorf("debugger: %v", err)
	}

	// add tab completion to terminal
	dbg.term.RegisterTabCompletion(commandline.NewTabCompletion(debuggerCommands))

	// create rewind system
	dbg.Rewind, err = rewind.NewRewind(dbg, dbg)
	if err != nil {
		return nil, curated.Errorf("debugger: %v", err)
	}

	// add reflection system to the GUI
	dbg.ref = reflection.NewReflector(dbg.vcs)
	if r, ok := dbg.gui.(reflection.Broker); ok {
		dbg.ref.AddRenderer(r.GetReflectionRenderer())
	}

	// add counter to rewind system
	dbg.counter = counter.NewCounter(dbg.vcs)
	dbg.Rewind.AddTimelineCounter(dbg.counter)

	// adding TV frame triggers in setMode(). what the TV triggers on depending
	// on the mode for performance reasons (eg. no reflection required in
	// playmode)

	// add audio tracker
	dbg.Tracker = tracker.NewTracker(dbg)
	dbg.vcs.TIA.Audio.SetTracker(dbg.Tracker)

	// add plug monitor
	dbg.vcs.RIOT.Ports.AttachPlugMonitor(dbg)

	// set fps cap
	dbg.vcs.TV.SetFPSCap(*opts.FpsCap)
	dbg.gui.SetFeature(gui.ReqMonitorSync, *opts.FpsCap)

	// initialise terminal
	err = dbg.term.Initialise()
	if err != nil {
		return nil, curated.Errorf("debugger: %v", err)
	}

	return dbg, nil
}

// VCS implements the emulation.Emulation interface.
func (dbg *Debugger) VCS() emulation.VCS {
	return dbg.vcs
}

// TV implements the emulation.Emulation interface.
func (dbg *Debugger) TV() emulation.TV {
	return dbg.vcs.TV
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

// Mode implements the emulation.Emulation interface.
func (dbg *Debugger) Mode() emulation.Mode {
	return dbg.mode.Load().(emulation.Mode)
}

// set the emulation state
//
// * if the state is the Paused or Running state consider using
// debugger.SetFeature(ReqSetPause) even from within the debugger package
// (SetFeature() puts the request on the RawEvent Queue meaning it will be
// inserted in the input loop correctly)
func (dbg *Debugger) setState(state emulation.State) {
	dbg.setStateQuiet(state, false)
}

// same as setState but with quiet argument, to indicate that EmulationEvent
// should not be issued to the gui.
//
// * see setState() comment, although debugger.SetFeature(ReqSetPause) will
// always be "noisy"
func (dbg *Debugger) setStateQuiet(state emulation.State, quiet bool) {
	if state == emulation.Rewinding {
		dbg.vcs.Mem.Cart.BreakpointsDisable(true)
		dbg.endPlayback()
		dbg.endRecording()
		dbg.endComparison()

		// it is thought that bots are okay to enter the rewinding state. this
		// might not be true in all future cases.
	} else {
		dbg.vcs.Mem.Cart.BreakpointsDisable(false)
	}

	err := dbg.vcs.TV.SetEmulationState(state)
	if err != nil {
		logger.Log("debugger", err.Error())
	}
	if dbg.ref != nil {
		dbg.ref.SetEmulationState(state)
	}

	prevState := dbg.State()
	dbg.state.Store(state)

	if !quiet && dbg.Mode() == emulation.ModePlay {
		switch state {
		case emulation.Initialising:
			dbg.gui.SetFeature(gui.ReqEmulationEvent, emulation.EventInitialising)
		case emulation.Paused:
			dbg.gui.SetFeature(gui.ReqEmulationEvent, emulation.EventPause)
		case emulation.Running:
			if prevState > emulation.Initialising {
				dbg.gui.SetFeature(gui.ReqEmulationEvent, emulation.EventRun)
			}
		}
	}
}

// set the emulation mode
//
// * consider using debugger.SetFeature(ReqSetMode) even from within the
// debugger package (SetFeature() puts the request on the RawEvent Queue
// meaning it will be inserted in the input loop correctly)
func (dbg *Debugger) setMode(mode emulation.Mode) error {
	if dbg.Mode() == mode {
		return nil
	}

	// don't stop the recording, playback, comparison or bot sub-systems on
	// change of mode

	// if there is a halting condition that is not allowed in playmode (see
	// targets type) then do not change the emulation mode
	//
	// however, because the user has asked to switch to playmode we should
	// cause the debugger mode to run until the halting condition is matched
	// (which we know will occur in the almost immediate future)
	if mode == emulation.ModePlay && !dbg.halting.allowPlaymode() {
		if dbg.Mode() == emulation.ModeDebugger {
			dbg.runUntilHalt = true
			dbg.continueEmulation = true
		}
		return nil
	}

	prevMode := dbg.Mode()
	dbg.mode.Store(mode)

	// notify gui of change
	err := dbg.gui.SetFeature(gui.ReqSetEmulationMode, mode)
	if err != nil {
		return err
	}

	// remove all triggers that we work with in the debugger. we'll add the
	// one's we want depending on playmode.
	//
	// note that we don't remove *every* frame trigger because other sub-systems
	// might have added their own.
	dbg.vcs.TV.RemoveFrameTrigger(dbg.ref)
	dbg.vcs.TV.RemoveFrameTrigger(dbg.Rewind)
	dbg.vcs.TV.RemoveFrameTrigger(dbg.counter)

	// swtich mode and make sure emulation is in correct state. we say that
	// emulation is always running when entering playmode and always paused
	// when entering debug mode.
	//
	// * the reason for this is simplicity. if we allow playmode to begin
	// paused for example it complicates how we render the screen (see sdlimgui
	// screen.go)

	switch dbg.Mode() {
	case emulation.ModePlay:
		dbg.vcs.TV.AddFrameTrigger(dbg.Rewind)
		dbg.vcs.TV.AddFrameTrigger(dbg.counter)

		// simple detection of whether cartridge is ejected when switching to
		// playmode. if it is ejected then open ROM selected.
		if dbg.Mode() == emulation.ModePlay && dbg.vcs.Mem.Cart.IsEjected() {
			err = dbg.forceROMSelector()
			if err != nil {
				return curated.Errorf("debugger: %v", err)
			}
		} else {
			dbg.setState(emulation.Running)
		}

	case emulation.ModeDebugger:
		dbg.vcs.TV.AddFrameTrigger(dbg.Rewind)
		dbg.vcs.TV.AddFrameTrigger(dbg.ref)
		dbg.vcs.TV.AddFrameTrigger(dbg.counter)
		dbg.setState(emulation.Paused)

		// debugger needs knowledge about previous frames (via the reflector)
		// if we're moving from playmode. also we want to make sure we end on
		// an instruction boundary.
		//
		// playmode will always break on an instruction boundary but without
		// catchupEndAdj we will always enter the debugger on the last cycle of
		// an instruction. although correct in terms of coordinates, is
		// confusing.
		if prevMode == emulation.ModePlay {
			dbg.catchupEndAdj = true
			dbg.RerunLastNFrames(2)
		}
	default:
		return curated.Errorf("emulation mode not supported: %s", mode)
	}

	return nil
}

// End cleans up any resources that may be dangling.
func (dbg *Debugger) end() {
	dbg.endPlayback()
	dbg.endRecording()
	dbg.endComparison()
	dbg.bots.Quit()

	dbg.vcs.End()

	defer dbg.term.CleanUp()

	// set ending state
	err := dbg.gui.SetFeature(gui.ReqEnd)
	if err != nil {
		logger.Log("debugger", err.Error())
	}

	// save preferences
	err = dbg.Prefs.Save()
	if err != nil {
		logger.Log("debugger", err.Error())
	}
}

// StartInDebugMode starts the emulation with the debugger activated.
func (dbg *Debugger) StartInDebugMode(filename string) error {
	// set running flag as early as possible
	dbg.running = true

	var err error
	var cartload cartridgeloader.Loader

	if filename == "" {
		cartload = cartridgeloader.Loader{}
	} else {
		cartload, err = cartridgeloader.NewLoader(filename, *dbg.opts.Mapping)
		if err != nil {
			return curated.Errorf("debugger: %v", err)
		}
	}

	err = dbg.attachCartridge(cartload)
	if err != nil {
		return curated.Errorf("debugger: %v", err)
	}

	err = dbg.insertPeripheralsOnStartup(*dbg.opts.Left, *dbg.opts.Right)
	if err != nil {
		return curated.Errorf("debugger: %v", err)
	}

	err = dbg.setMode(emulation.ModeDebugger)
	if err != nil {
		return curated.Errorf("debugger: %v", err)
	}

	// intialisation script because we're in debugger mode
	if *dbg.opts.InitScript != "" {
		scr, err := script.RescribeScript(*dbg.opts.InitScript)
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

	defer dbg.end()
	err = dbg.run()
	if err != nil {
		if curated.Has(err, terminal.UserQuit) {
			return nil
		}
		return curated.Errorf("debugger: %v", err)
	}

	return nil
}

func (dbg *Debugger) insertPeripheralsOnStartup(left string, right string) error {
	dbg.term.Silence(true)
	defer dbg.term.Silence(false)

	err := dbg.parseCommand(fmt.Sprintf("PERIPHERAL LEFT %s", left), false, false)
	if err != nil {
		return err
	}
	err = dbg.parseCommand(fmt.Sprintf("PERIPHERAL RIGHT %s", right), false, false)
	if err != nil {
		return err
	}
	return nil
}

// StartInPlaymode starts the emulation ready for game-play.
func (dbg *Debugger) StartInPlayMode(filename string) error {
	// set running flag as early as possible
	dbg.running = true

	var err error
	var cartload cartridgeloader.Loader

	if filename == "" {
		cartload = cartridgeloader.Loader{}
	} else {
		cartload, err = cartridgeloader.NewLoader(filename, *dbg.opts.Mapping)
		if err != nil {
			return curated.Errorf("debugger: %v", err)
		}
	}

	err = recorder.IsPlaybackFile(filename)
	if err != nil {
		if !curated.Is(err, recorder.NotAPlaybackFile) {
			return curated.Errorf("debugger: %v", err)
		}

		err = dbg.attachCartridge(cartload)
		if err != nil {
			return curated.Errorf("debugger: %v", err)
		}

		err = dbg.insertPeripheralsOnStartup(*dbg.opts.Left, *dbg.opts.Right)
		if err != nil {
			return curated.Errorf("debugger: %v", err)
		}

		// apply patch if requested. note that this will be in addition to any
		// patches applied during setup.AttachCartridge
		if *dbg.opts.PatchFile != "" {
			_, err := patch.CartridgeMemory(dbg.vcs.Mem.Cart, *dbg.opts.PatchFile)
			if err != nil {
				return curated.Errorf("debugger: %v", err)
			}
		}

		// record wav file
		if *dbg.opts.Wav {
			fn := unique.Filename("audio", cartload.ShortName())
			ww, err := wavwriter.NewWavWriter(fn)
			if err != nil {
				return curated.Errorf("debugger: %v", err)
			}
			dbg.vcs.TV.AddAudioMixer(ww)
		}

		// record gameplay
		if *dbg.opts.Record {
			dbg.startRecording(cartload.ShortName())
		}
	} else {
		if *dbg.opts.Record {
			return curated.Errorf("debugger: cannot make a new recording using a playback file")
		}

		dbg.startPlayback(filename)
	}

	err = dbg.startComparison(*dbg.opts.ComparisonROM, *dbg.opts.ComparisonPrefs)
	if err != nil {
		return curated.Errorf("debugger: %v", err)
	}

	err = dbg.setMode(emulation.ModePlay)
	if err != nil {
		return curated.Errorf("debugger: %v", err)
	}

	// wait a very short time to give window time to open. this would be better
	// and more consistently achieved with the help of a synchronisation channel
	<-time.After(250 * time.Millisecond)

	defer dbg.end()
	err = dbg.run()
	if err != nil {
		if curated.Has(err, terminal.UserQuit) {
			return nil
		}
		return curated.Errorf("debugger: %v", err)
	}

	return nil
}

func (dbg *Debugger) run() error {
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
		switch dbg.Mode() {
		case emulation.ModePlay:
			err := dbg.playLoop()
			if err != nil {
				// if we ever encounter a cartridge ejected error in playmode
				// then simply open up the ROM selector
				if curated.Has(err, cartridge.Ejected) {
					err = dbg.forceROMSelector()
					if err != nil {
						return curated.Errorf("debugger: %v", err)
					}
				} else {
					return curated.Errorf("debugger: %v", err)
				}
			}
		case emulation.ModeDebugger:
			switch dbg.State() {
			case emulation.Running:
				dbg.runUntilHalt = true
				dbg.continueEmulation = true
			case emulation.Paused:
				dbg.haltImmediately = true
			case emulation.Rewinding:
			default:
				return curated.Errorf("emulation state not supported on *start* of debugging loop: %s", dbg.State())
			}

			err := dbg.inputLoop(dbg.term, false)
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
			} else if dbg.State() == emulation.Ending {
				dbg.running = false
			}
		default:
			return curated.Errorf("emulation mode not supported: %s", dbg.mode)
		}
	}

	// make sure any cartridge loader has been finished with
	if dbg.loader != nil {
		err := dbg.loader.Close()
		if err != nil {
			return curated.Errorf("debugger: %v", err)
		}
	}

	return nil
}

// reset of VCS should go through this function to makes sure debugger is reset
// accordingly also. note that debugging features (breakpoints, etc.) are not
// reset.
//
// the newCartridge flag will cause breakpoints, traces, etc. to be reset
// as well. it is sometimes appropriate to reset these (eg. on new cartridge
// insert)
func (dbg *Debugger) reset(newCartridge bool) error {
	err := dbg.vcs.Reset()
	if err != nil {
		return err
	}
	dbg.Rewind.Reset()
	dbg.Tracker.Reset()

	// reset other debugger properties that might not make sense for a new cartride
	if newCartridge {
		dbg.halting.breakpoints.clear()
		dbg.halting.traps.clear()
		dbg.halting.watches.clear()
		dbg.traces.clear()
	}

	dbg.liveDisasmEntry = &disassembly.Entry{Result: execution.Result{Final: true}}
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
//
// if the new cartridge loader has the same filename as the previous loader
// then reset() is called with a newCartridge argument of false.
func (dbg *Debugger) attachCartridge(cartload cartridgeloader.Loader) (e error) {
	// is this a new cartridge we're loading. value is used for dbg.reset()
	newCartridge := dbg.loader == nil || cartload.Filename != dbg.loader.Filename

	// stop optional sub-systems that shouldn't survive a new cartridge insertion
	dbg.endPlayback()
	dbg.endRecording()
	dbg.endComparison()
	dbg.bots.Quit()

	// attching a cartridge implies the initialise state
	dbg.setState(emulation.Initialising)

	// set state after initialisation according to the emulation mode
	defer func() {
		switch dbg.Mode() {
		case emulation.ModeDebugger:
			if dbg.runUntilHalt && e == nil {
				dbg.setState(emulation.Running)
			} else {
				dbg.setState(emulation.Paused)
			}
		case emulation.ModePlay:
			dbg.setState(emulation.Running)
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
				if dbg.multiload >= 0 {
					logger.Logf("debugger", "forcing supercharger multiload (%#02x)", uint8(dbg.multiload))
					dbg.vcs.Mem.Poke(supercharger.MutliloadByteAddress, uint8(dbg.multiload))
				}

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
				dbg.liveDisasmEntry.Result.Final = true

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
				err := dbg.gui.SetFeature(gui.ReqCartridgeEvent, mapper.EventSuperchargerSoundloadStarted)
				if err != nil {
					return err
				}
			case mapper.EventSuperchargerSoundloadEnded:
				err := dbg.gui.SetFeature(gui.ReqCartridgeEvent, mapper.EventSuperchargerSoundloadEnded)
				if err != nil {
					return err
				}

				// !!TODO: it would be nice to see partial disassemblies of supercharger tapes
				// during loading. not completely necessary I don't think, but it would be
				// nice to have.
				err = dbg.Disasm.FromMemory()
				if err != nil {
					return err
				}

				return dbg.vcs.TV.Reset(true)
			case mapper.EventSuperchargerSoundloadRewind:
				err := dbg.gui.SetFeature(gui.ReqCartridgeEvent, mapper.EventSuperchargerSoundloadRewind)
				if err != nil {
					return err
				}
			default:
				logger.Logf("debugger", "unhandled hook event for supercharger (%v)", event)
			}
		} else if _, ok := cart.(*plusrom.PlusROM); ok {
			switch event {
			case mapper.EventPlusROMInserted:
				if dbg.vcs.Instance.Prefs.PlusROM.NewInstallation {
					err := dbg.gui.SetFeature(gui.ReqPlusROMFirstInstallation)
					if err != nil {
						if !curated.Is(err, gui.UnsupportedGuiFeature) {
							return curated.Errorf("debugger: %v", err)
						}
					}
				}
			case mapper.EventPlusROMNetwork:
				err := dbg.gui.SetFeature(gui.ReqCartridgeEvent, mapper.EventPlusROMNetwork)
				if err != nil {
					return err
				}
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

	// clear existing reflection and counter data
	dbg.ref.Clear()
	dbg.counter.Clear()

	err = dbg.Disasm.FromMemory()
	if err != nil {
		logger.Logf("debugger", err.Error())
	}

	coproc := dbg.vcs.Mem.Cart.GetCoProcBus()
	dbg.CoProcDisasm = coprocDisasm.NewDisassembly(dbg.vcs.TV, coproc)
	dbg.CoProcDev = coprocDev.NewDeveloper(filepath.Dir(cartload.Filename), coproc, dbg.vcs.TV)
	if dbg.CoProcDev != nil {
		dbg.vcs.TV.AddFrameTrigger(dbg.CoProcDev)
	}

	// make sure everything is reset after disassembly (including breakpoints, etc.)
	dbg.reset(newCartridge)

	// activate bot if possible
	feedback, err := dbg.bots.ActivateBot(dbg.vcs.Mem.Cart.Hash)
	if err != nil {
		logger.Logf("debugger", err.Error())
	}

	// always ReqBotFeedback. if feedback is nil then the bot features will be disbaled
	err = dbg.gui.SetFeature(gui.ReqBotFeedback, feedback)
	if err != nil {
		return curated.Errorf("debugger: %v", err)
	}

	// record the most filename as the most recent ROM loaded if appropriate
	if !dbg.vcs.Mem.Cart.IsEjected() {
		dbg.Prefs.RecentROM.Set(cartload.Filename)
	}

	return nil
}

func (dbg *Debugger) startRecording(cartShortName string) error {
	recording := unique.Filename("recording", cartShortName)

	var err error

	dbg.recorder, err = recorder.NewRecorder(recording, dbg.vcs)
	if err != nil {
		return err
	}

	return nil
}

func (dbg *Debugger) endRecording() {
	if dbg.recorder == nil {
		return
	}
	defer func() {
		dbg.recorder = nil
	}()

	err := dbg.recorder.End()
	if err != nil {
		logger.Logf("debugger", err.Error())
	}
}

func (dbg *Debugger) startPlayback(filename string) error {
	plb, err := recorder.NewPlayback(filename, *dbg.opts.PlaybackCheckROM)
	if err != nil {
		return err
	}

	err = dbg.attachCartridge(plb.CartLoad)
	if err != nil {
		return err
	}

	err = plb.AttachToVCSInput(dbg.vcs)
	if err != nil {
		return err
	}

	dbg.playback = plb

	return nil
}

func (dbg *Debugger) endPlayback() {
	if dbg.playback == nil {
		return
	}

	dbg.playback = nil
	dbg.vcs.Input.AttachPlayback(nil)
}

func (dbg *Debugger) startComparison(comparisonROM string, comparisonPrefs string) error {
	if comparisonROM == "" {
		return nil
	}

	// add any bespoke comparision prefs
	prefs.PushCommandLineStack(comparisonPrefs)

	var err error

	dbg.comparison, err = comparison.NewComparison(dbg.vcs)
	if err != nil {
		return err
	}
	dbg.gui.SetFeature(gui.ReqComparison, dbg.comparison.Render, dbg.comparison.DiffRender)

	cartload, err := cartridgeloader.NewLoader(comparisonROM, "AUTO")
	if err != nil {
		return err
	}

	dbg.comparison.CreateFromLoader(cartload)

	// check use of comparison prefs
	comparisonPrefs = prefs.PopCommandLineStack()
	if comparisonPrefs != "" {
		logger.Logf("debugger", "%s unused for comparison emulation", comparisonPrefs)
	}

	return nil
}

func (dbg *Debugger) endComparison() {
	if dbg.comparison == nil {
		return
	}

	dbg.comparison.Quit()
	dbg.comparison = nil
	dbg.gui.SetFeature(gui.ReqComparison, nil, nil)
}

func (dbg *Debugger) hotload() (e error) {
	// tell GUI that we're in the initialistion phase
	dbg.setState(emulation.Initialising)
	defer func() {
		if dbg.runUntilHalt && e == nil {
			dbg.setState(emulation.Running)
		} else {
			dbg.gui.SetFeature(gui.ReqEmulationEvent, emulation.EventPause)
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

	err = dbg.vcs.Mem.Cart.HotLoad(cartload)
	if err != nil {
		return err
	}

	dbg.loader = &cartload

	// disassemble newly attached cartridge
	err = dbg.Disasm.FromMemory()
	if err != nil {
		return err
	}

	dbg.vcs.TV.RemoveFrameTrigger(dbg.CoProcDev)

	coproc := dbg.vcs.Mem.Cart.GetCoProcBus()
	dbg.CoProcDisasm = coprocDisasm.NewDisassembly(dbg.vcs.TV, coproc)
	dbg.CoProcDev = coprocDev.NewDeveloper(filepath.Dir(cartload.Filename), coproc, dbg.vcs.TV)
	if dbg.CoProcDev != nil {
		dbg.vcs.TV.AddFrameTrigger(dbg.CoProcDev)
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

// Plugged implements the plugging.PlugMonitor interface.
func (dbg *Debugger) Plugged(port plugging.PortID, peripheral plugging.PeripheralID) {
	if dbg.vcs.Mem.Cart.IsEjected() {
		return
	}
	dbg.gui.SetFeature(gui.ReqControllerChange, port, peripheral)
}

// PushRawEvent onto the event queue.
//
// Used to ensure that the events are inserted into the emulation loop
// correctly.
func (dbg *Debugger) PushRawEvent(f func()) {
	select {
	case dbg.events.RawEvents <- f:
	default:
		logger.Log("debugger", "dropped raw event push")
	}
}

// PushRawEventImmediate is the same as PushRawEvent but handlers will return to the
// input loop for immediate action.
func (dbg *Debugger) PushRawEventImmediate(f func()) {
	select {
	case dbg.events.RawEventsImmediate <- f:
	default:
		logger.Log("debugger", "dropped raw event push (to return channel)")
	}
}
