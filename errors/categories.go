package errors

// list of error numbers
const (
	// Debugger
	InputEmpty Errno = iota
	CommandError
	SymbolsFileCannotOpen
	SymbolsFileError
	SymbolUnknown
	ScriptFileCannotOpen
	ScriptFileError
	ScriptRunError
	InvalidTarget
	CannotRecordState

	// Regression
	RegressionEntryExists
	RegressionEntryCollision
	RegressionEntryDoesNotExist
	RegressionEntryFail

	// CPU
	UnimplementedInstruction
	InvalidOpcode
	ProgramCounterCycled
	InvalidOperationMidInstruction

	// Memory
	UnservicedChipWrite
	UnknownRegisterName
	UnreadableAddress
	UnwritableAddress
	UnrecognisedAddress
	UnPokeableAddress

	// Cartridges
	CartridgeFileError
	CartridgeUnsupported
	CartridgeMissing
	CartridgeNoSuchBank

	// TV
	UnknownTVRequest
	SDLTV
	ImageTV
	DigestTV

	// Peripherals
	NoControllersFound
)
