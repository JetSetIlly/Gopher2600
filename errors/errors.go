package errors

import "fmt"

// list of sub-systems used when defining errors
const (
	CategoryDebugger   = 0
	CategoryVCS        = 64
	CategoryCPU        = 128
	CategoryMemory     = 192
	CategoryTIA        = 256
	CategoryRIOT       = 320
	CategoryTV         = 384
	CategoryController = 448
)

// list of error numbers
const (
	// Debugger
	NoSymbolsFile = CategoryDebugger + iota
	SymbolsFileError

	// VCS

	// CPU
	UnimplementedInstruction = CategoryCPU + iota
	NullInstruction

	// Memory
	UnservicedChipWrite = CategoryMemory + iota
	UnknownRegisterName
	UnreadableAddress

	// TIA

	// RIOT

	// TV

	// Controller
	NoControllersFound = CategoryController + iota
)

var messages = map[int]string{
	// Debugger
	NoSymbolsFile:    "no symbols file for %s",
	SymbolsFileError: "error processing symbols file (%s)",

	// VCS

	// CPU
	UnimplementedInstruction: "unimplemented instruction (%0#x)",
	NullInstruction:          "unimplemented instruction (0xff)",

	// Memory
	UnservicedChipWrite: "chip memory write signal has not been serviced since previous write (%s)",
	UnknownRegisterName: "can't find register name (%s) in list of read addreses in %s memory",
	UnreadableAddress:   "memory location is not readable",

	// TIA

	// RIOT

	// TV

	// Controller
	NoControllersFound: "no controllers found",
}

// Values is the type used to specify arguments for a GopherError
type Values []interface{}

// GopherError is the error type used by Gopher2600
type GopherError struct {
	Errno  int
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
