package disassembly

import (
	"fmt"
	"gopher2600/errors"
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/memory"
	"gopher2600/symbols"
	"io"
)

const bankMask = 0x0fff

type bank [bankMask + 1]Entry

// Disassembly represents the annotated disassembly of a 6502 binary
type Disassembly struct {
	Cart *memory.Cartridge

	// simply anlysis of the cartridge
	selfModifyingCode bool
	interrupts        bool
	forcedRTS         bool

	// symbols used to build disassembly output
	Symtable *symbols.Table

	// linear is the decoding of every possible address in the cartridge (see
	// linear.go for fuller commentary)
	linear []bank

	// flow is the decoding of cartridge addresses that follow the flow from
	// the address pointed to by the reset address of the cartridge. (see
	// flow.go for fuller commentary)
	flow []bank
}

func (dsm Disassembly) String() string {
	return fmt.Sprintf("non-cart JMPs: %v\ninterrupts: %v\nforced RTS: %v\n", dsm.selfModifyingCode, dsm.interrupts, dsm.forcedRTS)
}

// Get returns the disassembled entry at the specified bank/address
func (dsm Disassembly) Get(bank int, address uint16) (Entry, bool) {
	entry := dsm.linear[bank][address&bankMask]
	return entry, entry.IsInstruction()
}

// Dump writes the entire disassembly to the write interface
func (dsm *Disassembly) Dump(output io.Writer) {
	for bank := 0; bank < dsm.Cart.NumBanks; bank++ {
		output.Write([]byte(fmt.Sprintf("--- bank %d ---\n", bank)))

		for i := range dsm.flow[bank] {
			if entry := dsm.flow[bank][i]; entry.instructionDefinition != nil {
				output.Write([]byte(entry.instruction))
				output.Write([]byte("\n"))
			}
		}
	}
}

// FromCartrige initialises a new partial emulation and returns a
// disassembly from the supplied cartridge filename. - useful for one-shot
// disassemblies, like the gopher2600 "disasm" mode
func FromCartrige(cartridgeFilename string) (*Disassembly, error) {
	// ignore errors caused by loading of symbols table
	symtable, err := symbols.ReadSymbolsFile(cartridgeFilename)
	if err != nil {
		symtable = symbols.StandardSymbolTable()
	}

	cart := memory.NewCart()

	err = cart.Attach(cartridgeFilename)
	if err != nil {
		return nil, errors.NewFormattedError(errors.DisasmError, err)
	}

	dsm := new(Disassembly)
	err = dsm.FromMemory(cart, symtable)
	if err != nil {
		return dsm, err // no need to wrap error
	}

	return dsm, nil
}

// FromMemory disassembles an existing instance of cartridge memory using a
// cpu with no flow control
func (dsm *Disassembly) FromMemory(cart *memory.Cartridge, symtable *symbols.Table) error {
	dsm.Cart = cart
	dsm.Symtable = symtable
	dsm.flow = make([]bank, dsm.Cart.NumBanks)
	dsm.linear = make([]bank, dsm.Cart.NumBanks)

	// create new memory
	mem, err := newDisasmMemory(dsm.Cart)
	if err != nil {
		return errors.NewFormattedError(errors.DisasmError, err)
	}

	// create a new NoFlowControl CPU to help disassemble memory
	mc, err := cpu.NewCPU(mem)
	if err != nil {
		return errors.NewFormattedError(errors.DisasmError, err)
	}
	mc.NoFlowControl = true

	// disassemble linearly

	// make sure we're in the starting bank - at the beginning of the
	// disassembly and at the end
	dsm.Cart.BankSwitch(0)
	defer dsm.Cart.BankSwitch(0)

	err = mc.LoadPCIndirect(memory.AddressReset)
	if err != nil {
		return errors.NewFormattedError(errors.DisasmError, err)
	}
	err = dsm.linearDisassembly(mc)
	if err != nil {
		return errors.NewFormattedError(errors.DisasmError, err)
	}

	// disassemble as best we can with (manual) flow control

	mc.Reset()

	err = mc.LoadPCIndirect(memory.AddressReset)
	if err != nil {
		return errors.NewFormattedError(errors.DisasmError, err)
	}
	err = dsm.flowDisassembly(mc)
	if err != nil {
		return errors.NewFormattedError(errors.DisasmError, err)
	}

	return nil
}
