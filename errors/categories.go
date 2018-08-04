package errors

// list of error numbers
const (
	// Debugger
	SymbolsFileCannotOpen Errno = iota
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
