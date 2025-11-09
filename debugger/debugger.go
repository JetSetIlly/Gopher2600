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
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/jetsetilly/gopher2600/bots/wrangler"
	"github.com/jetsetilly/gopher2600/cartridgeloader"
	"github.com/jetsetilly/gopher2600/comparison"
	"github.com/jetsetilly/gopher2600/coprocessor"
	coproc_dev "github.com/jetsetilly/gopher2600/coprocessor/developer"
	coproc_dwarf "github.com/jetsetilly/gopher2600/coprocessor/developer/dwarf"
	coproc_disasm "github.com/jetsetilly/gopher2600/coprocessor/disassembly"
	"github.com/jetsetilly/gopher2600/debugger/dbgmem"
	"github.com/jetsetilly/gopher2600/debugger/govern"
	"github.com/jetsetilly/gopher2600/debugger/script"
	"github.com/jetsetilly/gopher2600/debugger/terminal"
	"github.com/jetsetilly/gopher2600/debugger/terminal/commandline"
	"github.com/jetsetilly/gopher2600/disassembly"
	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/gui"
	"github.com/jetsetilly/gopher2600/hardware"
	"github.com/jetsetilly/gopher2600/hardware/cpu/execution"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/supercharger"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/hardware/television"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/hardware/tia/audio"
	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/macro"
	"github.com/jetsetilly/gopher2600/notifications"
	"github.com/jetsetilly/gopher2600/patch"
	"github.com/jetsetilly/gopher2600/prefs"
	"github.com/jetsetilly/gopher2600/preview"
	"github.com/jetsetilly/gopher2600/properties"
	"github.com/jetsetilly/gopher2600/recorder"
	"github.com/jetsetilly/gopher2600/reflection"
	"github.com/jetsetilly/gopher2600/reflection/counter"
	"github.com/jetsetilly/gopher2600/resources/unique"
	"github.com/jetsetilly/gopher2600/rewind"
	"github.com/jetsetilly/gopher2600/setup"
	"github.com/jetsetilly/gopher2600/tracker"
	"github.com/jetsetilly/gopher2600/userinput"
	"github.com/jetsetilly/gopher2600/video"
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
	state    atomic.Value // emulation.State
	subState atomic.Value // emulation.RewindingSubState

	// reference to emulated hardware. this pointer never changes through the
	// life of the emulation even though the hardware may change and the
	// components may change (during rewind for example)
	vcs *hardware.VCS

	// keep a reference to the current cartridgeloader to make sure Close() is called
	cartload *cartridgeloader.Loader

	// preview emulation is used to gather information about a ROM before
	// running it fully
	preview *preview.Emulation

	// gameplay recorder/playback
	recorder *recorder.Recorder
	playback *recorder.Playback

	// macro (only one allowed for the time being)
	macro *macro.Macro

	// comparison emulator
	comparison *comparison.Comparison

	// GUI, terminal and controllers
	gui         gui.GUI
	term        terminal.Terminal
	controllers *userinput.Controllers

	// stella.Properties file support
	Properties properties.Properties

	// bots coordinator
	bots *wrangler.Bots

	// when reading input from the terminal there are other events
	// that need to be monitored
	events *terminal.ReadEvents

	// how often the events field should be checked
	readEventsPulse *time.Ticker

	// cartridge disassembly
	//
	// * allocated when entering debugger mode
	Disasm       *disassembly.Disassembly
	CoProcDisasm coproc_disasm.Disassembly
	CoProcDev    coproc_dev.Developer

	// the live disassembly entry. updated every CPU step or on halt (which may
	// be mid instruction). it is also updated by the LAST command when the
	// debugger is in the CLOCK quantum
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

	// the live disassembly entry. updated every CPU step or on halt (which may
	// be mid instruction). it is also updated by the LAST command when the
	// debugger is in the CLOCK quantum
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

	// Quantum to use when stepping/running
	quantum atomic.Value // govern.Quantum

	// record user input to a script file
	scriptScribe script.Scribe

	// the Rewind system stores and restores machine state
	Rewind *rewind.Rewind

	// the amount we rewind by is dependent on how fast the mouse wheel is
	// moving or for how long the keyboard (or gamepad bumpers) have been
	// depressed.
	//
	// for keyboard rewinding, the amount to rewind by increases for as long as
	// the key-combo (gamepad bumper) is being held. the amount is reset when
	// the key-combo/bumper is released
	rewindKeyboardAccumulation int

	// when rewinding by mouse wheel we just use the delta information from the
	// input device. however, we might miss mousewheel events during the
	// debugger catchup loop. the rewindMouseWheelAccumulation value allows us
	// to accumulate the delta value until we have the opportunity to issue
	// another rewind instruction
	rewindMouseWheelAccumulation int

	// audio tracker stores audio state over time
	Tracker *tracker.Tracker

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

	// the context in which the catchup loop is running
	catchupContext catchupContext

	// the debugger catchup loop will end on a video cycle if necessary. this
	// is what we want in most situations but occasionally it is useful to stop
	// on an instruction boundary. catchupEndAdj will ensure that the debugger
	// halts on an instruction boundary
	//
	// the value will reset to false at the end of a catchup loop
	catchupEndAdj bool

	// when switching to debug mode from CartYield() we don't want to rewind
	// and rerun the emulation because the condition which caused the yield
	// might not happen again
	//
	// this is the case when a breakpoint is triggered as a result of user
	// input. currently, user input is not reinserted into the emulation on the
	// rerun. it should be possible to do given the input system's flexibility
	// but for now we'll just use this switch
	//
	// the only feature we lose with this is incomplete reflection information
	// immediately after switching to the debugger
	noRewindOnSwitchToDebugger bool
}

// CreateUserInterface is used to initialise the user interface used by the
// emulation. It returns an instance of both the GUI and Terminal interfaces
// in the repsective packages.
type CreateUserInterface func(*Debugger) (gui.GUI, terminal.Terminal, error)

// NewDebugger creates and initialises everything required for a new debugging
// session.
//
// It should be followed up with a call to AddUserInterface() and call the
// Start() method to actually begin the emulation.
func NewDebugger(opts CommandLineOptions, create CreateUserInterface) (*Debugger, error) {
	dbg := &Debugger{
		// copy of the arguments
		opts: opts,

		// the ticker to indicate whether we should check for events in the
		// inputLoop. this used to be set to 50ms but that is way too long and
		// introduces a three frame lag to input devices. put another way, it
		// results in the emulation missing crucial input information. this is
		// particularly noticeable with paddle input because the paddle
		// resistance value (see paddle.go file in the peripherals/controllers
		// package) is only updated every three frames, causing the paddle to be
		// effectively running three times slower than the screen
		readEventsPulse: time.NewTicker(1 * time.Millisecond),
	}

	// set atomics to defaults values. if we don't do this we can cause panics
	// due to the GUI asking for values before we've had a chance to set them
	dbg.state.Store(govern.EmulatorStart)
	dbg.subState.Store(govern.RewindingBackwards)
	dbg.mode.Store(govern.ModeNone)
	dbg.quantum.Store(govern.QuantumInstruction)

	var err error

	// creat a new television. this will be used during the initialisation of
	// the VCS and not referred to directly again
	tv, err := television.NewTelevision(opts.Spec)
	if err != nil {
		return nil, fmt.Errorf("debugger: %w", err)
	}

	// create a new VCS instance
	dbg.vcs, err = hardware.NewVCS(environment.MainEmulation, tv, dbg, nil)
	if err != nil {
		return nil, fmt.Errorf("debugger: %w", err)
	}

	// create userinput/controllers handler
	dbg.controllers = userinput.NewControllers(dbg.vcs.Input)

	// create bot coordinator
	dbg.bots = wrangler.NewBots(dbg.vcs.Input, dbg.vcs.TV)

	// stella.pro support
	dbg.Properties, err = properties.Load()
	if err != nil {
		logger.Log(logger.Allow, "debugger", err)
	}

	// create preview emulation
	dbg.preview, err = preview.NewEmulation(dbg.vcs.Env.Prefs, tv.GetResetSpecID())
	if err != nil {
		return nil, fmt.Errorf("debugger: %w", err)
	}

	// set up debugging interface to memory
	dbg.dbgmem = &dbgmem.DbgMem{
		VCS: dbg.vcs,
	}

	// create a new disassembly instance
	dbg.Disasm, dbg.dbgmem.Sym, err = disassembly.NewDisassembly(dbg.vcs)
	if err != nil {
		return nil, fmt.Errorf("debugger: %w", err)
	}

	// create new coprocessor developer/disassembly instances
	dbg.CoProcDisasm = coproc_disasm.NewDisassembly(dbg.vcs.TV)
	dbg.CoProcDev = coproc_dev.NewDeveloper(dbg, dbg.vcs.TV)
	dbg.vcs.TV.AddFrameTrigger(&dbg.CoProcDev)

	// create a minimal lastResult for initialisation
	dbg.liveDisasmEntry = &disassembly.Entry{Result: execution.Result{Final: true}}

	// halting coordination
	dbg.halting, err = newHaltCoordination(dbg)
	if err != nil {
		return nil, fmt.Errorf("debugger: %w", err)
	}

	// add TV to halting coordination
	dbg.vcs.TV.AddDebugger(dbg.halting)

	// traces
	dbg.traces = newTraces(dbg)

	// make synchronisation channels. PushedFunctions can be pushed thick and
	// fast and the channel queue should be pretty lengthy to prevent dropped
	// events (see PushFunction()).
	dbg.events = &terminal.ReadEvents{
		UserInput:        make(chan userinput.Event, 10),
		UserInputHandler: dbg.userInputHandler,
		Signal:           make(chan os.Signal, 1),
		SignalHandler: func(sig os.Signal) error {
			switch sig {
			case syscall.SIGHUP:
				return terminal.UserReload
			case syscall.SIGINT:
				return terminal.UserInterrupt
			case syscall.SIGQUIT:
				return terminal.UserQuit
			case syscall.SIGKILL:
				// we're unlikely to receive the kill signal, it normally being
				// intercepted by the terminal, but in case we do we treat it
				// like the QUIT signal
				return terminal.UserQuit
			default:
			}
			return nil
		},
		PushedFunction:          make(chan func(), 4096),
		PushedFunctionImmediate: make(chan func(), 4096),
	}

	// connect signals to dbg.events.Signal channel. we include the Kill signal
	// but the chances are it'll never be seen
	signal.Notify(dbg.events.Signal, os.Interrupt, os.Kill, syscall.SIGHUP, syscall.SIGQUIT)

	// allocate memory for user input
	dbg.input = make([]byte, 255)

	// create GUI
	dbg.gui, dbg.term, err = create(dbg)
	if err != nil {
		return nil, fmt.Errorf("debugger: %w", err)
	}

	// add tab completion to terminal
	dbg.term.RegisterTabCompletion(commandline.NewTabCompletion(debuggerCommands))

	// create rewind system
	dbg.Rewind, err = rewind.NewRewind(dbg, dbg)
	if err != nil {
		return nil, fmt.Errorf("debugger: %w", err)
	}

	// attach rewind system as an event recorder
	dbg.vcs.Input.AddRecorder(dbg.Rewind)

	// add reflection system to the GUI
	dbg.ref = reflection.NewReflector(dbg.vcs)
	if r, ok := dbg.gui.(reflection.Broker); ok {
		dbg.ref.AddRenderer(r.GetReflectionRenderer())
	}

	// add counter to rewind system
	dbg.counter = counter.NewCounter(dbg.vcs)
	dbg.Rewind.AddTimelineCounter(dbg.counter)

	// add disassembly to rewind system. this is for the sequential disassembly listing
	dbg.Rewind.AddSplicer(dbg.Disasm)

	// adding TV frame triggers in setMode(). what the TV triggers on depending
	// on the mode for performance reasons (eg. no reflection required in
	// playmode)

	// add audio tracker
	dbg.Tracker = tracker.NewTracker(dbg, dbg.vcs.TV, dbg.Rewind)
	dbg.vcs.TIA.Audio.SetTracker(dbg.Tracker)

	// add plug monitor
	dbg.vcs.RIOT.Ports.AttachPlugMonitor(dbg)

	// set fps cap
	dbg.vcs.TV.SetFPSLimit(opts.FpsCap)

	// initialise terminal
	err = dbg.term.Initialise()
	if err != nil {
		return nil, fmt.Errorf("debugger: %w", err)
	}

	// add extensions to debugger commands
	debuggerCommands.AddExtension("mapper specific", dbg.vcs.Mem.Cart)
	debuggerCommands.AddExtension("symbol", &dbg.Disasm.Sym)
	debuggerCommands.AddExtension("read symbol", &dbg.Disasm.Sym)
	debuggerCommands.AddExtension("write symbol", &dbg.Disasm.Sym)
	debuggerCommands.AddExtension("label", &dbg.Disasm.Sym)

	return dbg, nil
}

// VCS implements the emulation.Emulation interface.
func (dbg *Debugger) VCS() *hardware.VCS {
	return dbg.vcs
}

// TV implements the emulation.Emulation interface.
func (dbg *Debugger) TV() *television.Television {
	return dbg.vcs.TV
}

// Debugger implements the emulation.Emulation interface.
func (dbg *Debugger) Debugger() *Debugger {
	return dbg
}

// UserInput implements the emulation.Emulation interface.
func (dbg *Debugger) UserInput() chan userinput.Event {
	return dbg.events.UserInput
}

// Mode implements the emulation.Emulation interface.
func (dbg *Debugger) Quantum() govern.Quantum {
	return dbg.quantum.Load().(govern.Quantum)
}

// set the quantum state
func (dbg *Debugger) setQuantum(quantum govern.Quantum) {
	dbg.quantum.Store(quantum)
}

// State implements the emulation.Emulation interface.
func (dbg *Debugger) State() govern.State {
	return dbg.state.Load().(govern.State)
}

// SubState implements the emulation.Emulation interface.
func (dbg *Debugger) SubState() govern.SubState {
	return dbg.subState.Load().(govern.SubState)
}

// Mode implements the emulation.Emulation interface.
func (dbg *Debugger) Mode() govern.Mode {
	return dbg.mode.Load().(govern.Mode)
}

// set the emulation state
func (dbg *Debugger) setState(state govern.State, subState govern.SubState) {
	// intentionally panic if state/sub-state combination is not allowed
	if !govern.StateIntegrity(state, subState) {
		panic(fmt.Sprintf("illegal sub-state (%s) for %s state (prev state: %s)",
			subState, state,
			dbg.state.Load().(govern.State),
		))
	}

	if state == govern.Rewinding {
		dbg.endPlayback()
		dbg.endRecording()
		dbg.endComparison()

		// coprocessor disassembly is an inherently slow operation particuarly
		// for StrongARM type ROMs
		dbg.CoProcDisasm.Inhibit(true)
	} else {
		// uninhibit coprocessor disassembly
		dbg.CoProcDisasm.Inhibit(false)
	}

	err := dbg.vcs.TV.SetEmulationState(state)
	if err != nil {
		logger.Log(logger.Allow, "debugger", err)
	}
	if dbg.ref != nil {
		dbg.ref.SetEmulationState(state)
	}
	dbg.CoProcDev.SetEmulationState(state)
	dbg.Rewind.SetEmulationState(state)

	dbg.state.Store(state)
	dbg.subState.Store(subState)
}

// set the emulation mode
func (dbg *Debugger) setMode(mode govern.Mode) error {
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
	if mode == govern.ModePlay && !dbg.halting.allowPlaymode() {
		if dbg.Mode() == govern.ModeDebugger {
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
	case govern.ModePlay:
		dbg.vcs.TV.AddFrameTrigger(dbg.Rewind)
		dbg.vcs.TV.AddFrameTrigger(dbg.counter)
		dbg.setState(govern.Running, govern.Normal)

	case govern.ModeDebugger:
		dbg.vcs.TV.AddFrameTrigger(dbg.Rewind)
		dbg.vcs.TV.AddFrameTrigger(dbg.ref)
		dbg.vcs.TV.AddFrameTrigger(dbg.counter)
		dbg.setState(govern.Paused, govern.Normal)

		// debugger needs knowledge about previous frames (via the reflector)
		// if we're moving from playmode. also we want to make sure we end on
		// an instruction boundary.
		//
		// playmode will always break on an instruction boundary but without
		// catchupEndAdj we will always enter the debugger on the last cycle of
		// an instruction. although correct in terms of coordinates, is
		// confusing.
		if prevMode == govern.ModePlay {
			if !dbg.noRewindOnSwitchToDebugger {
				dbg.catchupEndAdj = true
				dbg.RerunLastNFrames(2, nil)
			}
		}
	default:
		return fmt.Errorf("emulation mode not supported: %s", mode)
	}

	return nil
}

// End cleans up any resources that may be dangling.
func (dbg *Debugger) end() {
	dbg.endPlayback()
	dbg.endRecording()
	dbg.endComparison()
	if dbg.macro != nil {
		dbg.macro.Quit()
	}
	dbg.bots.Quit()

	dbg.vcs.End()

	defer dbg.term.CleanUp()

	// set ending state
	err := dbg.gui.SetFeature(gui.ReqEnd)
	if err != nil {
		logger.Log(logger.Allow, "debugger", err)
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
		cartload, err = cartridgeloader.NewLoaderFromFilename(filename, dbg.opts.Mapping, dbg.opts.Bank, dbg.Properties)
		if err != nil {
			return fmt.Errorf("debugger: %w", err)
		}
	}

	// cartload is should be passed to attachCartridge() almost immediately. the
	// closure of cartload will then be handled for us
	err = dbg.attachCartridge(cartload)
	if err != nil {
		return fmt.Errorf("debugger: %w", err)
	}

	err = dbg.setPeripheralsOnStartup()
	if err != nil {
		return fmt.Errorf("debugger: %w", err)
	}

	err = dbg.setMode(govern.ModeDebugger)
	if err != nil {
		return fmt.Errorf("debugger: %w", err)
	}

	// intialisation script because we're in debugger mode
	if dbg.opts.InitScript != "" {
		scr, err := script.RescribeScript(dbg.opts.InitScript)
		if err == nil {
			dbg.term.Silence(true)
			err = dbg.inputLoop(scr, false)
			if err != nil {
				dbg.term.Silence(false)
				return err
			}

			dbg.term.Silence(false)
		}
	}

	defer dbg.end()

	err = dbg.run()
	if err != nil {
		if errors.Is(err, terminal.UserSignal) {
			return nil
		}
		return fmt.Errorf("debugger: %w", err)
	}

	return nil
}

func (dbg *Debugger) setPeripheralsOnStartup() error {
	dbg.term.Silence(true)
	defer dbg.term.Silence(false)

	err := dbg.parseCommand(fmt.Sprintf("PERIPHERAL LEFT %s", dbg.opts.Left), false, false)
	if err != nil {
		return err
	}
	err = dbg.parseCommand(fmt.Sprintf("PERIPHERAL RIGHT %s", dbg.opts.Right), false, false)
	if err != nil {
		return err
	}

	if dbg.opts.SwapPorts {
		err = dbg.parseCommand(fmt.Sprintf("PERIPHERAL SWAP"), false, false)
		if err != nil {
			return err
		}
	}

	return nil
}

// StartInPlaymode starts the emulation ready for game-play.
func (dbg *Debugger) StartInPlayMode(filename string) error {
	// set running flag as early as possible
	dbg.running = true

	err := recorder.IsPlaybackFile(filename)
	if err != nil {
		if !errors.Is(err, recorder.NotAPlaybackFile) {
			return fmt.Errorf("debugger: %w", err)
		}

		var cartload cartridgeloader.Loader

		if filename == "" {
			cartload = cartridgeloader.Loader{}
		} else {
			cartload, err = cartridgeloader.NewLoaderFromFilename(filename, dbg.opts.Mapping, dbg.opts.Bank, dbg.Properties)
			if err != nil {
				return fmt.Errorf("debugger: %w", err)
			}
		}

		// cartload is should be passed to attachCartridge() almost immediately. the
		// closure of cartload will then be handled for us
		err = dbg.attachCartridge(cartload)
		if err != nil {
			return fmt.Errorf("debugger: %w", err)
		}

		err = dbg.setPeripheralsOnStartup()
		if err != nil {
			return fmt.Errorf("debugger: %w", err)
		}

		// apply patch if requested. note that this will be in addition to any
		// patches applied during setup.AttachCartridge
		if dbg.opts.PatchFile != "" {
			err := patch.CartridgeMemoryFromFile(dbg.vcs.Mem.Cart, dbg.opts.PatchFile)
			if err != nil {
				if errors.Is(err, patch.PatchFileNotFound) {
					logger.Log(logger.Allow, "debugger", err)
				} else {
					return fmt.Errorf("debugger: %w", err)
				}
			} else {
				logger.Logf(logger.Allow, "debugger", "cartridge patched: %s", dbg.opts.PatchFile)
			}
		}

	} else {
		err = dbg.startPlayback(filename)
		if err != nil {
			return fmt.Errorf("debugger: %w", err)
		}
	}

	// record gameplay
	if dbg.opts.Record {
		dbg.startRecording()
	}

	// record wav file
	if dbg.opts.Wav {
		fn := unique.Filename("audio", dbg.cartload.Name)
		ww, err := wavwriter.NewWavWriter(fn, audio.AverageSampleFreq)
		if err != nil {
			return fmt.Errorf("debugger: %w", err)
		}
		dbg.vcs.TV.AddAudioMixer(ww)
	}

	// record video
	if dbg.opts.Video {
		var endFrame int
		if dbg.playback != nil {
			endFrame = dbg.playback.EndFrame()
		}
		err := dbg.gui.SetFeature(gui.ReqVideoRecord, true, video.Session{
			Log:       os.Stdout,
			LastFrame: endFrame,
			Profile:   video.ProfileFast,
		})
		if err != nil {
			return fmt.Errorf("debugger: %w", err)
		}
	}

	if dbg.opts.Macro != "" {
		dbg.macro, err = macro.NewMacro(dbg.opts.Macro, dbg, dbg.vcs.Input, dbg.vcs.TV, dbg.gui)
		if err != nil {
			return fmt.Errorf("debugger: %w", err)
		}
	}

	err = dbg.startComparison(dbg.opts.ComparisonROM, dbg.opts.ComparisonPrefs)
	if err != nil {
		return fmt.Errorf("debugger: %w", err)
	}

	err = dbg.setMode(govern.ModePlay)
	if err != nil {
		return fmt.Errorf("debugger: %w", err)
	}

	// wait a very short time to give window time to open. this would be better
	// and more consistently achieved with the help of a synchronisation channel
	<-time.After(250 * time.Millisecond)

	defer dbg.end()

	if dbg.macro != nil {
		dbg.macro.Run()
	}

	err = dbg.run()
	if err != nil {
		if errors.Is(err, terminal.UserSignal) {
			return nil
		}
		return fmt.Errorf("debugger: %w", err)
	}

	return nil
}

// CartYield implements the coprocessor.CartYieldHook interface.
func (dbg *Debugger) CartYield(yield coprocessor.CoProcYield) coprocessor.YieldHookResponse {
	// if the emulator wants to quit we need to return true to instruct the
	// cartridge to return to the main loop immediately
	if !dbg.running {
		return coprocessor.YieldHookEnd
	}

	// resolve deferred yield
	if dbg.halting.deferredCartridgeYield {
		dbg.halting.deferredCartridgeYield = false
		dbg.halting.cartridgeYield = yield
		return coprocessor.YieldHookEnd
	}

	switch yield.Type {
	case coprocessor.YieldProgramEnded:
		// expected reason for CDF and DPC+ cartridges
		return coprocessor.YieldHookContinue

	case coprocessor.YieldSyncWithVCS:
		// expected reason for ACE and ELF cartridges
		return coprocessor.YieldHookContinue
	}

	// if emulation is in the intialisation state then we cause coprocessor
	// execution to end unless it's a memory or access erorr
	//
	// this is an area that's likely to change. it's of particular interest to
	// ACE and ELF ROMs in which the coprocessor is run very early in order to
	// retrive the 6507 reset address
	//
	// a deferred YeildHookEnd might be a better option
	if dbg.State() == govern.Initialising {
		dbg.halting.deferredCartridgeYield = true
		return coprocessor.YieldHookContinue
	}

	dbg.halting.cartridgeYield = yield
	dbg.continueEmulation = dbg.halting.check()

	switch dbg.Mode() {
	case govern.ModePlay:
		dbg.noRewindOnSwitchToDebugger = true
		dbg.setMode(govern.ModeDebugger)
		dbg.noRewindOnSwitchToDebugger = false
	case govern.ModeDebugger:
		dbg.inputLoop(dbg.term, true)
	}

	return coprocessor.YieldHookEnd
}

func (dbg *Debugger) run() error {
	// end script recording gracefully. this way we don't have to worry too
	// hard about script scribes
	defer func() {
		err := dbg.scriptScribe.EndSession()
		if err != nil {
			logger.Log(logger.Allow, "debugger", err)
		}
	}()

	// make sure any cartridge loader has been finished with
	defer func() {
		if dbg.cartload != nil {
			err := dbg.cartload.Close()
			if err != nil {
				logger.Log(logger.Allow, "debugger", err)
			}
		}
	}()

	// inputloop will continue until debugger is to be terminated
	for dbg.running {
		switch dbg.Mode() {
		case govern.ModePlay:
			err := dbg.playLoop()
			if err != nil {
				if errors.Is(err, terminal.UserReload) {
					err = dbg.reloadCartridge()
					if err != nil {
						logger.Log(logger.Allow, "debugger", err)
					}
				} else {
					return err
				}
			}

		case govern.ModeDebugger:
			switch dbg.State() {
			case govern.Running:
				dbg.runUntilHalt = true
				dbg.continueEmulation = true
			case govern.Paused:
				dbg.haltImmediately = true
			case govern.Rewinding:
			default:
				return fmt.Errorf("emulation state not supported on *start* of debugging loop: %s", dbg.State())
			}

			err := dbg.inputLoop(dbg.term, false)
			if err != nil {
				// there is no special handling for the cartridge ejected error,
				// unlike in play mode
				if errors.Is(err, terminal.UserReload) {
					dbg.reloadCartridge()
				} else {
					return err
				}
			}

		default:
			return fmt.Errorf("emulation mode not supported: %s", dbg.mode)
		}

		// handle inputLoopRestart and any on-restart function
		if dbg.unwindLoopRestart != nil {
			err := dbg.unwindLoopRestart()
			if err != nil {
				return err
			}
			dbg.unwindLoopRestart = nil
		} else if dbg.State() == govern.Ending {
			dbg.running = false
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

	// it is important that we reset the cpuBoundaryLastInstruction with the reset TV coordinates
	// because the zero value of the TelevisionCoords type is not the starting position of the TV
	dbg.cpuBoundaryLastInstruction = dbg.vcs.TV.GetCoords()

	dbg.ClearHaltReason()
	dbg.Rewind.Reset()
	dbg.Tracker.Reset()

	// reset other debugger properties that might not make sense for a new cartride
	if newCartridge {
		dbg.halting.breakpoints.clear()
		dbg.halting.traps.clear()
		dbg.halting.watches.clear()
		dbg.traces.clear()
	}

	dbg.Disasm.Reset()
	dbg.liveBankInfo = mapper.BankInfo{}
	dbg.liveDisasmEntry = &disassembly.Entry{Result: execution.Result{Final: true}}
	return nil
}

// PushNotify implements the notifications.Notify interface
func (dbg *Debugger) PushNotify(notice notifications.Notice, data ...string) error {
	dbg.PushFunction(func() {
		dbg.Notify(notice, data...)
	})
	return nil
}

// Notify implements the notifications.Notify interface
func (dbg *Debugger) Notify(notice notifications.Notice, data ...string) error {
	switch notice {

	case notifications.NotifySuperchargerFastload:
		// the supercharger ROM will eventually start execution from the PC
		// address given in the supercharger file

		// the interrupted CPU means it never got a chance to
		// finalise the result. we force that here by simply
		// setting the Final flag to true.
		dbg.vcs.CPU.LastResult.Final = true

		// we've already obtained the disassembled lastResult so we
		// need to change the final flag there too
		dbg.liveDisasmEntry.Result.Final = true

		// force multiload value for supercharger fastload
		if dbg.opts.Multiload >= 0 {
			dbg.vcs.Mem.Poke(supercharger.MutliloadByteAddress, uint8(dbg.opts.Multiload))
		}

		// call commit function to complete tape loading procedure
		fastload := dbg.vcs.Mem.Cart.GetSuperchargerFastLoad()
		if fastload == nil {
			return fmt.Errorf("NotifySuperchargerFastloadEnded sent from a non Supercharger fastload cartridge")
		}
		err := fastload.Fastload(dbg.vcs.CPU, dbg.vcs.Mem.RAM, dbg.vcs.RIOT.Timer)
		if err != nil {
			return err
		}

		// (re)disassemble memory on TapeLoaded error signal
		err = dbg.Disasm.FromMemory()
		if err != nil {
			return err
		}
	case notifications.NotifySuperchargerSoundloadStarted:
		// force multiload value for supercharger soundload
		if dbg.opts.Multiload >= 0 {
			dbg.vcs.Mem.Poke(supercharger.MutliloadByteAddress, uint8(dbg.opts.Multiload))
		}

		err := dbg.gui.SetFeature(gui.ReqNotification, notifications.NotifySuperchargerSoundloadStarted)
		if err != nil {
			return err
		}
	case notifications.NotifySuperchargerSoundloadEnded:
		err := dbg.gui.SetFeature(gui.ReqNotification, notifications.NotifySuperchargerSoundloadEnded)
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
	case notifications.NotifySuperchargerSoundloadRewind:
		err := dbg.gui.SetFeature(gui.ReqNotification, notifications.NotifySuperchargerSoundloadRewind)
		if err != nil {
			return err
		}
	case notifications.NotifyPlusROMNewInstall:
		err := dbg.gui.SetFeature(gui.ReqNotification, notifications.NotifyPlusROMNewInstall)
		if err != nil {
			return err
		}
	case notifications.NotifyPlusROMNetwork:
		err := dbg.gui.SetFeature(gui.ReqNotification, notifications.NotifyPlusROMNetwork)
		if err != nil {
			return err
		}
	case notifications.NotifyElfUndefinedSymbols:
		err := dbg.gui.SetFeature(gui.ReqNotification, notifications.NotifyElfUndefinedSymbols)
		if err != nil {
			return err
		}
	case notifications.NotifyMovieCartStarted:
		return dbg.vcs.TV.Reset(true)
	case notifications.NotifyAtariVoxSubtitle:
		err := dbg.gui.SetFeature(gui.ReqNotification, notifications.NotifyAtariVoxSubtitle, data[0])
		if err != nil {
			return err
		}
	default:
		logger.Logf(logger.Allow, "debugger", "unhandled notification for plusrom (%v)", notice)
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
//
// if the new cartridge loader has the same filename as the previous loader
// then reset() is called with a newCartridge argument of false.
//
// VERY IMPORTANT that the supplied cartload is assigned to dbg.cartload before
// returning from the function
func (dbg *Debugger) attachCartridge(cartload cartridgeloader.Loader) (e error) {
	// is this a new cartridge we're loading. value is used for dbg.reset()
	newCartridge := dbg.cartload == nil || cartload.Filename != dbg.cartload.Filename

	// stop optional sub-systems that shouldn't survive a new cartridge insertion
	dbg.endPlayback()
	dbg.endRecording()
	dbg.endComparison()
	dbg.bots.Quit()

	// attching a cartridge implies the initialise state
	dbg.setState(govern.Initialising, govern.Normal)

	// set state after initialisation according to the emulation mode
	defer func() {
		switch dbg.Mode() {
		case govern.ModeDebugger:
			if dbg.runUntilHalt && e == nil {
				dbg.setState(govern.Running, govern.Normal)
			} else {
				dbg.setState(govern.Paused, govern.Normal)
			}
		case govern.ModePlay:
			dbg.setState(govern.Running, govern.Normal)
		}
	}()

	// close any existing loader before continuing
	if dbg.cartload != nil {
		err := dbg.cartload.Close()
		if err != nil {
			logger.Log(logger.Allow, "debuger", err)
		}
	}
	dbg.cartload = &cartload

	attachHook := func() {
		dbg.CoProcDisasm.AttachCartridge(dbg.vcs.Mem.Cart)
		err := dbg.CoProcDev.AttachCartridge(dbg.vcs.Mem.Cart, cartload.Filename, dbg.opts.DWARF)
		if err != nil {
			logger.Log(logger.Allow, "debugger", err)
			if errors.Is(err, coproc_dwarf.UnsupportedDWARF) {
				err = dbg.gui.SetFeature(gui.ReqNotification, notifications.NotifyUnsupportedDWARF)
				if err != nil {
					logger.Log(logger.Allow, "debugger", err)
				}
			}
		}

		// attach current debugger as the yield hook for cartridge
		dbg.vcs.Mem.Cart.SetYieldHook(dbg)
	}

	// reset of vcs is implied with attach cartridge
	err := setup.AttachCartridge(dbg.vcs, cartload, attachHook)
	if err != nil && !errors.Is(err, cartridge.Ejected) {
		logger.Log(logger.Allow, "debugger", err)
		// an error has occurred so attach the ejected cartridge
		//
		// !TODO: a special error cartridge to make it more obvious what has happened
		if err := setup.AttachCartridge(dbg.vcs, cartridgeloader.Loader{}, nil); err != nil {
			return err
		}
	}

	// perform disassembly in the background
	dbg.Disasm.Background(cartload)

	// check for cartridge ejection. if the NoEject option is set then return error
	if dbg.opts.NoEject && dbg.vcs.Mem.Cart.IsEjected() {
		// if there is an error left over from the AttachCartridge() call
		// above, return that rather than "cartridge ejected"
		if err != nil {
			return err
		}
		return fmt.Errorf("cartridge ejected")
	}

	// clear existing reflection and counter data
	dbg.ref.Clear()
	dbg.counter.Clear()

	// make sure everything is reset after disassembly (including breakpoints, etc.)
	dbg.reset(newCartridge)

	// run preview emulation
	err = dbg.preview.Run(cartload)
	if err != nil {
		if !errors.Is(err, cartridgeloader.NoFilename) {
			return err
		}
	}

	// reset cartridge loader after using it in the preview
	cartload.Seek(0, io.SeekStart)

	// copy resizer from preview to main emulation
	if dbg.vcs.TV.IsAutoSpec() {
		dbg.vcs.TV.SetSpec(dbg.preview.Results().SpecID)
	}
	dbg.vcs.TV.SetResizer(dbg.preview.Results().Resizer)

	// activate bot if possible
	feedback, err := dbg.bots.ActivateBot(dbg.vcs.Mem.Cart.Hash)
	if err != nil {
		logger.Log(logger.Allow, "debugger", err)
	}

	// always ReqBotFeedback. if feedback is nil then the bot features will be disbaled
	err = dbg.gui.SetFeature(gui.ReqBotFeedback, feedback)
	if err != nil {
		return err
	}

	return nil
}

func (dbg *Debugger) startRecording() error {
	var err error
	var recording string
	if dbg.opts.RecordFilename == "" {
		recording = unique.Filename("recording", dbg.cartload.Name)
	} else {
		recording = dbg.opts.RecordFilename
	}
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
		logger.Log(logger.Allow, "debugger", err)
	}
}

func (dbg *Debugger) startPlayback(filename string) error {
	plb, err := recorder.NewPlayback(filename, dbg.opts.PlaybackIgnoreDigest)
	if err != nil {
		return err
	}

	// new cartridge loader using the information found in the playback file
	cartload, err := cartridgeloader.NewLoaderFromFilename(plb.Cartridge, "AUTO", "AUTO", dbg.Properties)
	if err != nil {
		return fmt.Errorf("debugger: %w", err)
	}

	// check hash of cartridge before continuing
	if dbg.opts.PlaybackCheckROM && cartload.HashSHA1 != plb.Hash {
		return fmt.Errorf("playback: unexpected hash")
	}

	// cartload is should be passed to attachCartridge() almost immediately. the
	// closure of cartload will then be handled for us
	err = dbg.attachCartridge(cartload)
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
	err = dbg.gui.SetFeature(gui.ReqComparison, dbg.comparison.Render, dbg.comparison.DiffRender, dbg.comparison.AudioDiff)
	if err != nil {
		return err
	}

	cartload, err := cartridgeloader.NewLoaderFromFilename(comparisonROM, "AUTO", "AUTO", dbg.Properties)
	if err != nil {
		return err
	}

	// cartload is passed to comparision.CreateFromLoader(). closure will be
	// handled from there when comparision emulation ends
	dbg.comparison.CreateFromLoader(cartload)

	// check use of comparison prefs
	comparisonPrefs = prefs.PopCommandLineStack()
	if comparisonPrefs != "" {
		logger.Logf(logger.Allow, "debugger", "%s unused for comparison emulation", comparisonPrefs)
	}

	return nil
}

func (dbg *Debugger) endComparison() {
	if dbg.comparison == nil {
		return
	}

	dbg.comparison.Quit()
	dbg.comparison = nil
	err := dbg.gui.SetFeature(gui.ReqComparison, nil, nil, nil)
	if err != nil {
		logger.Log(logger.Allow, "debugger", err)
	}
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
	err := dbg.gui.SetFeature(gui.ReqPeripheralPlugged, port, peripheral)
	if err != nil {
		logger.Log(logger.Allow, "debugger", err)
	}
}

func (dbg *Debugger) reloadCartridge() error {
	if dbg.cartload == nil {
		return nil
	}

	// reset macro to beginning
	if dbg.macro != nil {
		dbg.macro.Reset()
	}

	return dbg.insertCartridge(dbg.cartload.Filename)
}

// ReloadCartridge inserts the current cartridge and states the emulation over.
func (dbg *Debugger) ReloadCartridge() {
	dbg.events.Signal <- syscall.SIGHUP
}

func (dbg *Debugger) insertCartridge(filename string) error {
	if filename == "" {
		filename = dbg.cartload.Filename
	}

	cartload, err := cartridgeloader.NewLoaderFromFilename(filename, dbg.opts.Mapping, dbg.opts.Bank, dbg.Properties)
	if err != nil {
		return fmt.Errorf("debugger: %w", err)
	}

	// cartload is should be passed to attachCartridge() almost immediately. the
	// closure of cartload will then be handled for us
	err = dbg.attachCartridge(cartload)
	if err != nil {
		return fmt.Errorf("debugger: %w", err)
	}

	if dbg.forcedROMselection != nil {
		dbg.forcedROMselection <- true
	}

	return nil
}

// InsertCartridge into running emulation. If filename argument is empty the
// currently inserted cartridge will be reinserted.
//
// It should not be run directly from the emulation/debugger goroutine, use
// insertCartridge() for that
func (dbg *Debugger) InsertCartridge(filename string, done chan bool) {
	dbg.PushFunctionImmediate(func() {
		dbg.setState(govern.Initialising, govern.Normal)
		dbg.unwindLoop(func() error {
			err := dbg.insertCartridge(filename)
			done <- err == nil
			return err
		})
	})
}

// GetLiveDisasmEntry returns the formatted disasembly entry of the last CPU
// execution and the bank informations.String())
func (dbg *Debugger) GetLiveDisasmEntry() disassembly.Entry {
	if dbg.liveDisasmEntry == nil {
		return disassembly.Entry{}
	}

	return *dbg.liveDisasmEntry
}

// memoryProfile forces a garbage collection event and takes a runtime memory
// profile and saves it to the working directory
func (dbg *Debugger) memoryProfile() (string, error) {
	fn := unique.Filename("", dbg.cartload.Name)
	fn = fmt.Sprintf("%s_mem.profile", fn)

	f, err := os.Create(fn)
	if err != nil {
		return "", err
	}
	defer f.Close()

	runtime.GC()
	err = pprof.WriteHeapProfile(f)
	if err != nil {
		return "", err
	}
	return fn, nil
}
