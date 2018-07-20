package errors

import "fmt"

// Errno is used specified the specific error
type Errno int

// list of sub-systems used when defining errors
const (
	CategoryDebugger = iota * 8
	CategoryVCS
	CategoryCPU
	CategoryMemory
	CategoryTIA
	CategoryRIOT
	CategoryTV
	CategoryController
)

// list of error numbers
const (
	// Debugger
	NoSymbolsFile Errno = CategoryDebugger + iota
	SymbolsFileError
	UnknownSymbol

	// VCS

	// CPU
	UnimplementedInstruction Errno = CategoryCPU + iota
	NullInstruction
	ProgramCounterCycled

	// Memory
	UnservicedChipWrite Errno = CategoryMemory + iota
	UnknownRegisterName
	UnreadableAddress

	// TIA

	// RIOT

	// TV
	UnknownStateRequest
	UnknownCallbackRequest
	InvalidStateRequest

	// Controller
	NoControllersFound Errno = CategoryController + iota
)

var messages = map[Errno]string{
	// Debugger
	NoSymbolsFile:    "no symbols file for %s",
	SymbolsFileError: "error processing symbols file (%s)",
	UnknownSymbol:    "unrecognised symbol (%s)",

	// VCS

	// CPU
	UnimplementedInstruction: "unimplemented instruction (%0#x) at (%#04x)",
	NullInstruction:          "unimplemented instruction (0xff)",
	ProgramCounterCycled:     "program counter cycled back to 0x0000",

	// Memory
	UnservicedChipWrite: "chip memory write signal has not been serviced since previous write (%s)",
	UnknownRegisterName: "can't find register name (%s) in list of read addreses in %s memory",
	UnreadableAddress:   "memory location is not readable",

	// TIA

	// RIOT

	// TV
	UnknownStateRequest:    "TV does not support %v state",
	UnknownCallbackRequest: "TV does not support %v callback",
	InvalidStateRequest:    "state request for %v is currently invalid",

	// Controller
	NoControllersFound: "no controllers found",
}

// Values is the type used to specify arguments for a GopherError
type Values []interface{}

// GopherError is the error type used by Gopher2600
type GopherError struct {
	Errno  Errno
	Values Values
}

func (er GopherError) Error() string {
	return fmt.Sprintf(messages[er.Errno], er.Values...)
}

// Category returns the broad categorisation of a GopherError
func (er GopherError) Category() int {
	if er.Errno >= CategoryController {
		return CategoryController
	}
	if er.Errno >= CategoryTV {
		return CategoryTV
	}
	if er.Errno >= CategoryRIOT {
		return CategoryRIOT
	}
	if er.Errno >= CategoryTIA {
		return CategoryTIA
	}
	if er.Errno >= CategoryMemory {
		return CategoryMemory
	}
	if er.Errno >= CategoryCPU {
		return CategoryCPU
	}
	if er.Errno >= CategoryVCS {
		return CategoryVCS
	}
	return CategoryDebugger
}
