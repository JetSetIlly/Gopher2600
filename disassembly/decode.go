// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package disassembly

import (
	"fmt"

	"github.com/jetsetilly/gopher2600/disassembly/symbols"
	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/cpu/execution"
	"github.com/jetsetilly/gopher2600/hardware/cpu/instructions"
	"github.com/jetsetilly/gopher2600/hardware/memory/cartridge/mapper"
	"github.com/jetsetilly/gopher2600/hardware/memory/memorymap"
	"github.com/jetsetilly/gopher2600/hardware/television/coords"
	"github.com/jetsetilly/gopher2600/logger"
)

// IMPORTANT: all functions in this file should be called in the Dissasembly
// critical section

type decode struct {
	// symbols is thread safe
	sym *symbols.Symbols

	mc            *cpu.CPU
	mem           *disasmMemory
	disasmEntries DisasmEntries

	formatResult func(bank mapper.BankInfo, result execution.Result, level EntryLevel) *Entry
}

func (dec decode) GetCoords() coords.TelevisionCoords {
	return coords.TelevisionCoords{}
}

func newDecode(dsm *Disassembly, startingBank int, copiedBanks []mapper.BankContent) (*decode, error) {
	dec := decode{
		sym: &dsm.Sym,
	}

	dec.formatResult = func(bank mapper.BankInfo, result execution.Result, level EntryLevel) *Entry {
		return formatResult(dsm, dec, bank, result, level)
	}

	dec.mem = newDisasmMemory(startingBank, copiedBanks)
	if dec.mem == nil {
		return nil, fmt.Errorf("could not create memory for decoding")
	}
	dec.mc = cpu.NewCPU(dec.mem)
	dec.mc.NoFlowControl = true

	dec.disasmEntries.Entries = make([][]*Entry, len(copiedBanks))
	for b := range copiedBanks {
		dec.disasmEntries.Entries[b] = make([]*Entry, memorymap.CartridgeBits+1)
	}

	// basic decoding pass
	err := dec.decode()
	if err == nil {
		// bless those entries which we're reasonably sure are real instructions
		err = dec.bless()
		if err != nil {
			return nil, err
		}
	}

	return &dec, nil
}

func (dec *decode) bless() error {
	// bless from reset address for every bank
	for b := range dec.disasmEntries.Entries {
		// get reset address from starting bank, taking into account the
		// possibility that bank is smalled thank 4096 bytes
		resetVector := cpu.Reset & (uint16(len(dec.mem.banks[b].Data) - 1))
		resetAddr := (uint16(dec.mem.banks[b].Data[resetVector+1]) << 8) | uint16(dec.mem.banks[b].Data[resetVector])

		// make sure reset address is valid
		_, area := memorymap.MapAddress(resetAddr, true)
		if area == memorymap.Cartridge {
			_ = dec.blessSequence(b, resetAddr, true)
		}
	}

	moreBlessing := true
	attempts := 0

	for moreBlessing {
		moreBlessing = false

		// make sure we're not going bezerk. a limit of 10 is probably too high
		attempts++
		if attempts > 10 {
			break
		}

		// loop through every bank in the cartridge and collate a list of blessings
		// for the bank
		for b := range dec.disasmEntries.Entries {
			for i := range dec.disasmEntries.Entries[b] {
				e := dec.disasmEntries.Entries[b][i]

				// ignore any non-blessed entry
				if e.Level != EntryLevelBlessed {
					continue
				}

				// if instruction is a JMP or JSR then check the target address and
				// which banks it can feasibly be in. add every matching bank to
				// the blessings list
				//
				// an example of interbank JMP/JSR jumping is the beginning of the
				// HeMan ROM. From bank 7 the following jumps to bank 5.
				//
				//	$fa03 JMP $f7e8
				if e.Result.Defn.Operator == instructions.JMP || e.Result.Defn.Operator == instructions.JSR {
					jmpAddress, area := memorymap.MapAddress(e.Result.InstructionData, true)

					if area == memorymap.Cartridge {
						l := jmpTargets(dec.mem.banks, jmpAddress)

						for _, jb := range l {
							if dec.blessSequence(jb, jmpAddress, false) {
								if dec.blessSequence(jb, jmpAddress, true) {
									moreBlessing = true
								}
							}

							// add label to the symbols table
							dec.sym.AddLabelAuto(jb, jmpAddress)
						}
					}
				} else {
					// if instruction is a branch then add the address of the successful branch
					// to the blessing list. assumption here is that a branch will never branch
					// to another bank
					// !!TODO: find practical examples of interbank branch jumping
					if e.Result.Defn.IsBranch() {
						dec.mc.PC.Load(e.Result.Address)
						dec.mc.PC.Add(uint16(e.Result.Defn.Bytes))
						operand := e.Result.InstructionData
						if operand&0x0080 == 0x0080 {
							operand |= 0xff00
						}
						dec.mc.PC.Add(operand)

						pcAddress, area := memorymap.MapAddress(dec.mc.PC.Value(), true)

						if area == memorymap.Cartridge {
							if dec.blessSequence(b, pcAddress, false) {
								if dec.blessSequence(b, pcAddress, true) {
									moreBlessing = true
								}
							}

							// add label to the symbols table
							dec.sym.AddLabelAuto(b, pcAddress)
						}
					}
				}
			}
		}
	}

	// correct DASM symbols. DASM sometimes records that a label is at a
	// location adjacent to were it should be. I've only ever seen it such the
	// label is one address after where it should be but other variations may
	// exists
	//
	// the loop below works by:
	// 1) looking for an automatically labelled address (AUTO) that has been blessed
	// 2) if the address following that label has a DASM symbol then
	// 3) remove the AUTO label
	// and
	// 4) move the DASM label to the previously labelled with the AUTO label
	for b := range dec.disasmEntries.Entries {
		for i := range dec.disasmEntries.Entries[b] {
			if dec.disasmEntries.Entries[b][i].Level >= EntryLevelBlessed {
				addr := dec.disasmEntries.Entries[b][i].Result.Address
				if sym, ok := dec.sym.GetLabel(b, addr); ok && sym.Source == symbols.SourceAuto {
					if i < len(dec.disasmEntries.Entries[b]) {
						postAddr := dec.disasmEntries.Entries[b][i+1].Result.Address
						if postSym, ok := dec.sym.GetLabel(b, postAddr); ok && postSym.Source == symbols.SourceDASM {
							_ = dec.sym.RemoveLabel(symbols.SourceAuto, b, addr)
							_ = dec.sym.RemoveLabel(symbols.SourceDASM, b, postAddr)
							dec.sym.AddLabel(symbols.SourceDASM, b, addr, postSym.Symbol)
						}
					}
				}
			}
		}
	}

	// remove any auto-labels that have been added to entries that have not
	// been blessed (these labels were added speculatively).
	for b := range dec.disasmEntries.Entries {
		for i := range dec.disasmEntries.Entries[b] {
			if dec.disasmEntries.Entries[b][i].Level < EntryLevelBlessed {
				_ = dec.sym.RemoveLabel(symbols.SourceAuto, b, dec.disasmEntries.Entries[b][i].Result.Address)
			}
		}
	}

	return nil
}

// jmpTargets returns a list of banks that a JMP or JSR target address may
// feasibly be in.
func jmpTargets(copiedBanks []mapper.BankContent, jmpAddress uint16) []int {
	l := make([]int, 0, len(copiedBanks))

	// only the significant bits of the jmp and origin addresses are compared.
	// this is the easiest way of handling different cartridge mirrors
	jmpAddress &= memorymap.CartridgeBits

	// find banks which can be mapped to cover the supplied jmpAddress
	for _, b := range copiedBanks {
		for _, o := range b.Origins {
			o &= memorymap.CartridgeBits

			if jmpAddress >= o && jmpAddress <= o+uint16(len(b.Data)) {
				l = append(l, b.Number)
				break
			}
		}
	}

	return l
}

// returns false if sequence is false. generally, the function should be called
// twice for a bank/addr. once with commit set to false and then if the return
// value is true, called again with commit set to true.
//
// if commit is true without the sequence being sure, a log entry will be made
// in the event of mistake
//
// it does not matter if addr is a normalised or unormalised address.
func (dec *decode) blessSequence(bank int, addr uint16, commit bool) bool {
	// mask address into indexable range
	a := addr & memorymap.CartridgeBits

	hasCommitted := false

	// examine every entry from the starting point a. next entry determined in
	// program counter style (ie. address plus instruction byte count)
	//
	// sequence will stop if:
	//  . an unknown opcode has been encountered
	//  . if the end of cartridge data has been reached
	//  . there is already blessed instruction between this and the next entry
	//  . a flow control instruction is encountered (this is normal and expected)
	//  . an instruction looks like fill characters
	//
	// the following will stop the blessing but indicate that the blessing is
	// "correct" if the following have been encountered:
	//
	//  . an RTS instruction
	//  . a branch instruction
	//  . an interrupt
	for a < uint16(len(dec.disasmEntries.Entries[bank])) {
		instruction := dec.disasmEntries.Entries[bank][a]

		// end run if entry has already been blessed
		if instruction.Level == EntryLevelBlessed {
			// this is okay and expected so return true to indicate that the
			// blessing should continue
			return true
		}

		next := a + uint16(instruction.Result.ByteCount)

		// break if address has looped around
		if next > next&memorymap.CartridgeBits {
			if hasCommitted {
				logger.Logf(logger.Allow, "disassembly", "blessSequence has blessed an instruction in a false sequence. discovered at bank %d: %s", bank, instruction.String())
			}
			return false
		}

		// if an entry between this entry and the next has already been
		// blessed then this track is probably not correct.
		for i := a + 1; i < next; i++ {
			if instruction.Level == EntryLevelBlessed {
				if hasCommitted {
					logger.Logf(logger.Allow, "disassembly", "blessSequence has blessed an instruction in a false sequence. discovered at bank %d: %s", bank, instruction.String())
				}
				return false
			}
		}

		// do not bless decoded instructions that look like fill characters
		if instruction.Result.Defn.OpCode == 0xff && instruction.Result.InstructionData == 0xffff {
			if hasCommitted {
				logger.Logf(logger.Allow, "disassembly", "blessSequence has blessed an instruction in a false sequence. discovered at bank %d: %s", bank, instruction.String())
			}
			return false
		}

		// do not bless break instructions
		if instruction.Operator == "brk" {
			return false
		}

		// promote the entry
		if commit {
			hasCommitted = true
			instruction.Level = EntryLevelBlessed
		}

		// end the blessing sequence if we encountered an instruction that breaks
		// the flow with no possibility of resumption. In practical terms this
		// means JMP, RTS and Interrupt instructions.
		if instruction.Operator == "jmp" || instruction.Operator == "rts" ||
			instruction.Result.Defn.Effect == instructions.Interrupt {
			return true
		}

		// next address
		a = next
	}

	// reached end of the bank without encountering any other halt condition
	return false
}

func (dec *decode) decode() error {
	for _, bank := range dec.mem.banks {
		dec.mem.currentBank = bank.Number

		// try each bank in each possible segment - banks that are smaller than
		// the cartridge addressing range can often be mapped into different
		// "segments" of cartridge memory
		for _, origin := range bank.Origins {
			// make sure origin address is rooted correctly. we'll convert all
			// addresses to the preferred mirror at the end of the disassembly
			dec.mem.origin = (origin & memorymap.CartridgeBits) | memorymap.OriginCart

			// the memtop for the bank
			memtop := origin + uint16(len(bank.Data)) - 1

			// reset CPU for each bank/origin
			err := dec.mc.Reset(nil)
			if err != nil {
				return fmt.Errorf("decode: %w", err)
			}

			// loop over entire address space for cartridge. even then bank
			// sizes are smaller than the address space it makes things easier
			// later if we put a valid entry at every index. entries outside of
			// the bank space will have level == EntryLevelUnused
			for address := memorymap.OriginCart; address <= memorymap.MemtopCart; address++ {
				// continue if entry has already been decoded
				e := dec.disasmEntries.Entries[bank.Number][address&memorymap.CartridgeBits]
				if e != nil && e.Level > EntryLevelUnmappable {
					continue
				}

				// decide whether address is mappable or not. even if it isn't, we'll still go
				// through the decoding process so that we have a usuable entry at all points
				// in the disassembly. this simplifies future iterations.
				entryLevel := EntryLevelUnmappable
				if address >= origin && address <= memtop {
					entryLevel = EntryLevelDecoded
				}

				// execute instruction at address
				dec.mc.PC.Load(address)
				err := dec.mc.ExecuteInstruction(cpu.NilCycleCallback)
				if err != nil {
					return fmt.Errorf("decode: %w", err)
				}

				// error on invalid instruction execution
				if err = dec.mc.LastResult.IsValid(); err != nil {
					return fmt.Errorf("decode: %w", err)
				}

				// add entry to disassembly
				ent := dec.formatResult(mapper.BankInfo{Number: bank.Number}, dec.mc.LastResult, entryLevel)
				dec.disasmEntries.Entries[bank.Number][address&memorymap.CartridgeBits] = ent
			}
		}
	}

	// sanity checks
	for b := range dec.disasmEntries.Entries {
		for _, a := range dec.disasmEntries.Entries[b] {
			if a == nil {
				return fmt.Errorf("decode: not every address has been decoded")
			}
			if a.Level == EntryLevelUnmappable {
				if a.Result.Defn.OpCode != 0x00 {
					return fmt.Errorf("decode: an unmappable bank address [%#04x bank %d] has a non 0x00 opcode", a.Result.Address, b)
				}
			}
		}
	}

	return nil
}
