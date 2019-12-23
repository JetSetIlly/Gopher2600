package disassembly

import (
	"fmt"
	"gopher2600/cartridgeloader"
	"gopher2600/disassembly/display"
	"gopher2600/errors"
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/cpu/execution"
	"gopher2600/hardware/memory/addresses"
	"gopher2600/hardware/memory/cartridge"
	"gopher2600/symbols"
	"strings"
)

const disasmMask = 0x0fff

type bank [disasmMask + 1]*display.Instruction

// Disassembly represents the annotated disassembly of a 6507 binary
type Disassembly struct {
	cart *cartridge.Cartridge

	// discovered/inferred cartridge attributes
	nonCartJmps bool
	interrupts  bool
	forcedRTS   bool

	// symbols used to format disassembly output
	Symtable *symbols.Table

	// linear is the decoding of every possible address in the cartridge
	linear []bank

	// flow is the decoding of cartridge addresses that follow the flow from
	// the start address
	flow []bank

	// formatting information for all entries in the flow disassembly.
	// excluding the linear disassembly because false positives entries might
	// upset the formatting.
	Columns display.Columns
}

// Analysis returns a summary of anything interesting found during disassembly.
func (dsm Disassembly) Analysis() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("non-cart JMPs: %v\n", dsm.nonCartJmps))
	s.WriteString(fmt.Sprintf("interrupts: %v\n", dsm.interrupts))
	s.WriteString(fmt.Sprintf("forced RTS: %v\n", dsm.forcedRTS))
	return s.String()
}

// Get returns the disassembly at the specified bank/address
//
// function works best when the address definitely points to a valid
// instruction. This probably means during the execution of a the cartridge
// with proper flow control.
func (dsm Disassembly) Get(bank int, address uint16) (*display.Instruction, bool) {
	col := dsm.linear[bank][address&disasmMask]
	return col, col != nil
}

// FormatResult is a wrapper for the display.Format() function using the
// current symbol table
func (dsm Disassembly) FormatResult(result execution.Result) (*display.Instruction, error) {
	return display.Format(result, dsm.Symtable)
}

// FromCartridge initialises a new partial emulation and returns a
// disassembly from the supplied cartridge filename. - useful for one-shot
// disassemblies, like the gopher2600 "disasm" mode
func FromCartridge(cartload cartridgeloader.Loader) (*Disassembly, error) {
	// ignore errors caused by loading of symbols table - we always get a
	// standard symbols table even in the event of an error
	symtable, _ := symbols.ReadSymbolsFile(cartload.Filename)

	cart := cartridge.NewCartridge()

	err := cart.Attach(cartload)
	if err != nil {
		return nil, errors.New(errors.DisasmError, err)
	}

	dsm, err := FromMemory(cart, symtable)
	if err != nil {
		return nil, errors.New(errors.DisasmError, err)
	}

	return dsm, nil
}

// FromMemory disassembles an existing instance of cartridge memory using a
// cpu with no flow control.
func FromMemory(cart *cartridge.Cartridge, symtable *symbols.Table) (*Disassembly, error) {
	dsm := &Disassembly{}

	dsm.cart = cart
	dsm.Symtable = symtable
	dsm.flow = make([]bank, dsm.cart.NumBanks())
	dsm.linear = make([]bank, dsm.cart.NumBanks())

	// exit early if cartridge memory self reports as being ejected
	if dsm.cart.IsEjected() {
		return dsm, nil
	}

	// save cartridge state and defer at end of disassembly. this is necessary
	// because during the disassembly process we may changed mutable parts of
	// the cartridge (eg. extra RAM)
	state := dsm.cart.SaveState()
	defer dsm.cart.RestoreState(state)

	// put cart into its initial state
	dsm.cart.Initialise()

	// create new memory
	mem := &disasmMemory{cart: cart}

	// create a new NoFlowControl CPU to help disassemble memory
	mc, err := cpu.NewCPU(mem)
	if err != nil {
		return nil, errors.New(errors.DisasmError, err)
	}
	mc.NoFlowControl = true

	// disassemble linearly

	err = mc.LoadPCIndirect(addresses.Reset)
	if err != nil {
		return nil, errors.New(errors.DisasmError, err)
	}
	err = dsm.linearDisassembly(mc)
	if err != nil {
		return nil, errors.New(errors.DisasmError, err)
	}

	// disassemble as best we can with manual flow control

	mc.Reset()
	dsm.cart.Initialise()

	err = mc.LoadPCIndirect(addresses.Reset)
	if err != nil {
		return nil, errors.New(errors.DisasmError, err)
	}

	err = dsm.flowDisassembly(mc)
	if err != nil {
		return nil, errors.New(errors.DisasmError, err)
	}

	return dsm, nil
}
