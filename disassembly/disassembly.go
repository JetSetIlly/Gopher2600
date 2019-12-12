package disassembly

import (
	"fmt"
	"gopher2600/cartridgeloader"
	"gopher2600/errors"
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/memory/addresses"
	"gopher2600/hardware/memory/cartridge"
	"gopher2600/symbols"
	"io"
	"strings"
)

const disasmMask = 0x0fff

type bank [disasmMask + 1]Entry

// Disassembly represents the annotated disassembly of a 6507 binary
type Disassembly struct {
	cart *cartridge.Cartridge

	// discovered/inferred cartridge attributes
	nonCartJmps bool
	interrupts  bool
	forcedRTS   bool

	// symbols used to build disassembly output
	Symtable *symbols.Table

	// linear is the decoding of every possible address in the cartridge
	linear []bank

	// flow is the decoding of cartridge addresses that follow the flow from
	// the start address
	flow []bank
}

// Analysis returns a summary of anything interesting found during disassembly.
func (dsm Disassembly) Analysis() string {
	s := strings.Builder{}
	s.WriteString(fmt.Sprintf("non-cart JMPs: %v\n", dsm.nonCartJmps))
	s.WriteString(fmt.Sprintf("interrupts: %v\n", dsm.interrupts))
	s.WriteString(fmt.Sprintf("forced RTS: %v\n", dsm.forcedRTS))
	return s.String()
}

// Get returns the disassembled entry at the specified bank/address. This
// function works best when the address definitely points to a valid
// instruction. This probably means during the execution of a the cartridge
// with proper flow control.
func (dsm Disassembly) Get(bank int, address uint16) (Entry, bool) {
	entry := dsm.linear[bank][address&disasmMask]
	return entry, entry.IsInstruction()
}

// Dump writes the entire disassembly to the write interface
func (dsm *Disassembly) Dump(output io.Writer) {
	for bank := 0; bank < len(dsm.flow); bank++ {
		output.Write([]byte(fmt.Sprintf("--- bank %d ---\n", bank)))

		for i := range dsm.flow[bank] {
			if entry := dsm.flow[bank][i]; entry.instructionDefinition != nil {
				output.Write([]byte(entry.instruction))
				output.Write([]byte("\n"))
			}
		}
	}
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
