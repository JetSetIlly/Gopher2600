package errors

// list of error numbers
const (
	FatalError Errno = iota

	// sentinal
	UserInterrupt
	UserSuspend
	ScriptEnd
	PowerOff
	PeriphUnplugged
	TVOutOfSpec

	// program modes
	PlayError
	DebuggerError
	DisasmError
	PerformanceError

	// debugger
	ParserError
	ValidationError
	InvalidTarget
	CommandError
	TerminalError

	// script
	ScriptScribeError
	ScriptFileUnavailable
	ScriptFileError
	ScriptRunError

	// recorder
	RecordingError
	PlaybackError
	PlaybackHashError

	// regression
	RegressionDBError
	RegressionSetupError

	// symbols
	SymbolsFileUnavailable
	SymbolsFileError
	SymbolUnknown

	// vcs
	VCSError

	// cpu
	UnimplementedInstruction
	InvalidOpcode
	InvalidResult
	ProgramCounterCycled
	InvalidOperationMidInstruction

	// memory
	MemoryError
	UnreadableAddress
	UnwritableAddress
	UnpokeableAddress
	UnpeekableAddress
	UnrecognisedAddress

	// cartridges
	CartridgeFileError
	CartridgeFileUnavailable
	CartridgeError
	CartridgeEjected

	// peripherals
	PeriphHardwareUnavailable
	UnknownPeriphEvent

	// tv
	UnknownTVRequest
	StellaTelevision
	ImageTV
	DigestTV

	// gui
	UnknownGUIRequest
	SDL
)
