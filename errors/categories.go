package errors

// list of error numbers
const (
	// Debugger
	InputInvalidCommand Errno = iota
	InputTooManyArgs
	InputTooFewArgs
	InputEmpty
	SymbolsFileCannotOpen
	SymbolsFileError
	SymbolUnknown
	ScriptFileCannotOpen
	ScriptFileError
	InvalidTarget

	// CPU
	UnimplementedInstruction
	NullInstruction
	ProgramCounterCycled

	// Memory
	UnservicedChipWrite
	UnknownRegisterName
	UnreadableAddress
	UnwritableAddress
	UnrecognisedAddress

	// Cartridges
	CartridgeFileCannotOpen
	CartridgeFileError
	CartridgeInvalidSize

	// TV
	UnknownStateRequest
	UnknownCallbackRequest
	InvalidStateRequest

	// Peripherals
	NoControllersFound
)
