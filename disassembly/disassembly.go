package disassembly

import (
	"fmt"
	"gopher2600/hardware/cpu"
	"gopher2600/hardware/cpu/result"
	"gopher2600/hardware/memory"
	"gopher2600/symbols"
	"io"
)

// Disassembly represents the annotated disassembly of a 6502 binary
type Disassembly struct {
	Cart *memory.Cartridge

	// symbols used to build disassembly output
	Symtable *symbols.Table

	// linear is the decoding of every possible address in the cartridge (see
	// linear.go for fuller commentary)
	linear [](map[uint16]*result.Instruction)

	// flow is the decoding of cartridge addresses that follow the flow from
	// the address pointed to by the reset address of the cartridge. (see
	// flow.go for fuller commentary)
	flow [](map[uint16]*result.Instruction)

	// these variables are simply to note conditions that are sometimes
	// encountered during a flow disassembly. any holes in the flow disassembly
	// will be caused by one of the following
	selfModifyingCode bool
	interrupts        bool
	forcedRTS         bool
}

func (dsm Disassembly) String() string {
	return fmt.Sprintf("non-cart JMPs: %v\ninterrupts: %v\nforced RTS: %v\n", dsm.selfModifyingCode, dsm.interrupts, dsm.forcedRTS)
}

// GetLinear returns the disassembled entry at the specified bank/address
func (dsm Disassembly) GetLinear(bank int, address uint16) (*result.Instruction, bool) {
	v, ok := dsm.linear[bank][address&dsm.Cart.Memtop()]
	return v, ok
}

// GetFlow returns the disassembled entry at the specified bank/address
func (dsm Disassembly) GetFlow(bank int, address uint16) (*result.Instruction, bool) {
	v, ok := dsm.flow[bank][address&dsm.Cart.Memtop()]
	return v, ok
}

// putFlow stores a disassembled entry - returns false if entry already exists
func (dsm Disassembly) putFlow(bank int, result *result.Instruction) bool {
	if _, ok := dsm.GetFlow(bank, result.Address); ok {
		return false
	}
	dsm.flow[bank][result.Address&dsm.Cart.Memtop()] = result
	return true
}

// DumpFlow writes the entire disassembly to the write interface
func (dsm *Disassembly) DumpFlow(output io.Writer) {
	for bank := 0; bank < dsm.Cart.NumBanks; bank++ {
		output.Write([]byte(fmt.Sprintf("--- bank %d ---\n", bank)))
		for a := dsm.Cart.Origin(); a <= dsm.Cart.Memtop(); a++ {
			if dsm.flow[bank][a] != nil {
				output.Write([]byte(dsm.flow[bank][a].GetString(dsm.Symtable, result.StyleFull)))
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
		fmt.Println(err)
		symtable = symbols.StandardSymbolTable()
	}

	cart := memory.NewCart()

	err = cart.Attach(cartridgeFilename)
	if err != nil {
		return nil, err
	}

	dsm := new(Disassembly)
	err = dsm.FromMemory(cart, symtable)
	if err != nil {
		return dsm, err
	}

	return dsm, nil
}

// FromMemory disassembles an existing instance of cartridge memory using a
// cpu with no flow control
func (dsm *Disassembly) FromMemory(cart *memory.Cartridge, symtable *symbols.Table) error {
	dsm.Cart = cart
	dsm.Symtable = symtable
	dsm.flow = make([]map[uint16]*result.Instruction, dsm.Cart.NumBanks)
	dsm.linear = make([]map[uint16]*result.Instruction, dsm.Cart.NumBanks)

	// create new memory
	mem, err := newDisasmMemory(dsm.Cart)
	if err != nil {
		return err
	}

	// create a new non-branching CPU to disassemble memory
	mc, err := cpu.NewCPU(mem)
	if err != nil {
		return err
	}
	mc.NoFlowControl = true

	// allocate memory for disassembly
	for bank := 0; bank < dsm.Cart.NumBanks; bank++ {
		dsm.Cart.BankSwitch(bank)
		dsm.flow[bank] = make(map[uint16]*result.Instruction)
		dsm.linear[bank] = make(map[uint16]*result.Instruction)
	}

	// make sure we're in the starting bank - at the beginning of the
	// disassembly and at the end
	dsm.Cart.BankSwitch(0)
	defer dsm.Cart.BankSwitch(0)

	mc.LoadPCIndirect(memory.AddressReset)

	err = dsm.linearDisassembly(mc)
	if err != nil {
		return err
	}

	return dsm.flowDisassembly(mc)
}
