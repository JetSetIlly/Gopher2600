package errors

// list of error numbers
const (
	FatalError Errno = iota

	// sentinal
	UserInterrupt
	ScriptEnd
	PowerOff
	PeriphUnplugged
	TVOutOfSpec

	// program modes
	PlayError
	DebuggerError
	DisasmError
	FPSError

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
	RegressionFail

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
	BasicTelevision
	ImageTV
	DigestTV

	// gui
	UnknownGUIRequest
	SDL
)
