package errors

var messages = map[Errno]string{
	// panics
	PanicError: "FATALITY: %s: %s",

	// sentinals
	UserInterrupt:   "user interrupt",
	UserSuspend:     "user suspend",
	ScriptEnd:       "end of script (%s)",
	PowerOff:        "emulated machine has been powered off",
	PeriphUnplugged: "controller unplugged from '%s'",
	TVOutOfSpec:     "tv out of spec: %s",

	// program modes
	PlayError:        "error emulating vcs: %s",
	DebuggerError:    "error debugging vcs: %s",
	PerformanceError: "error during performance profiling: %s",
	DisasmError:      "error during disassembly: %s",

	// debugger
	ParserError:          "parser error: %s: %s (char %d)", // first placeholder is the command definition
	ValidationError:      "%s for %s",
	InvalidTarget:        "invalid target (%s)",
	CommandError:         "%s",
	TerminalError:        "%s",
	GUIEventError:        "%v",
	ReflectionNotRunning: "reflection process is not running",

	// script
	ScriptFileError:       "script error: %s",
	ScriptFileUnavailable: "script error: cannot open script file (%s)",
	ScriptRunError:        "script error: use of '%s' is not allowed in scripts [%s::%d]",
	ScriptScribeError:     "script scribe error: %s",

	// recorder
	RecordingError:    "controller recording error: %s",
	PlaybackError:     "controller playback error: %s",
	PlaybackHashError: "controller playback error: hash error: %s",

	// database
	DatabaseError:           "database error: %s",
	DatabaseSelectEmpty:     "database error: no selected entries",
	DatabaseKeyError:        "database error: no such key in database [%v]",
	DatabaseFileUnavailable: "database error: cannot open database (%s)",

	// regression
	RegressionError:         "regression test error: %s",
	RegressionFrameError:    "regression test error: frame entry: %s",
	RegressionPlaybackError: "regression test error: playback entry: %s",

	// setup
	SetupError:      "setup error: %s",
	SetupPanelError: "setup error: panel entry: %s",

	// symbols
	SymbolsFileError:       "symbols error: error processing symbols file: %s",
	SymbolsFileUnavailable: "symbols error: no symbols file for %s",
	SymbolUnknown:          "symbols error: unrecognised symbol (%s)",

	// cartridgeloader
	CartridgeLoader: "cartridge loading error: %s",

	// vcs
	VCSError: "vcs error: %s",

	// cpu
	UnimplementedInstruction:       "cpu error: unimplemented instruction (%0#x) at (%#04x)",
	InvalidOpcode:                  "cpu error: invalid opcode (%#04x)",
	InvalidResult:                  "cpu error: %s",
	ProgramCounterCycled:           "cpu error: program counter cycled back to 0x0000",
	InvalidOperationMidInstruction: "cpu error: invalid operation mid-instruction (%s)",

	// memory
	MemoryError:         "memory error: %s",
	UnreadableAddress:   "memory error: memory location is not readable (%#04x)",
	UnwritableAddress:   "memory error: memory location is not writable (%#04x)",
	UnpokeableAddress:   "memory error: cannot poke address (%v)",
	UnpeekableAddress:   "memory error: cannot peek address (%v)",
	UnrecognisedAddress: "memory error: address unrecognised (%v)",

	// cartridges
	CartridgeError:   "cartridge error: %s",
	CartridgeEjected: "cartridge error: no cartridge attached",

	// peripherals
	PeriphHardwareUnavailable: "peripheral error: controller hardware unavailable (%s)",
	UnknownPeriphEvent:        "peripheral error: %s: unsupported event (%v)",

	// television
	UnknownTVRequest: "television error: unsupported request (%v)",
	Television:       "television error: %s",

	// screen digest
	ScreenDigest: "television error: screendigest: %s",

	// gui
	UnsupportedGUIRequest: "gui error: unsupported request (%v)",
	SDL:                   "gui error: SDL: %s",
}
