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

	"github.com/jetsetilly/gopher2600/hardware/cpu"
	"github.com/jetsetilly/gopher2600/hardware/cpu/execution"
)

type quickDecode struct {
	mc  *cpu.CPU
	mem quickDecodeMemory
}

type quickDecodeMemory struct {
	origin uint16
	memtop uint16
	data   []uint8
}

func (mem *quickDecodeMemory) Read(address uint16) (uint8, error) {
	if address >= mem.origin && address <= mem.memtop {
		return mem.data[address-mem.origin], nil
	}
	return 0, nil
}

func (mem *quickDecodeMemory) Write(address uint16, data uint8) error {
	return nil
}

func newQuickDecode() *quickDecode {
	var dec quickDecode
	dec.mc = cpu.NewCPU(&dec.mem)
	dec.mc.NoFlowControl = true
	return &dec
}

func (dec *quickDecode) decode(address uint16, data []uint8, origin uint16) (execution.Result, error) {
	dec.mem.origin = origin
	dec.mem.memtop = origin + uint16(len(data))
	dec.mem.data = data[:]
	err := dec.mc.Reset(nil)
	if err != nil {
		return execution.Result{}, fmt.Errorf("quick decode: %w", err)
	}
	dec.mc.PC.Load(address)
	err = dec.mc.ExecuteInstruction(cpu.NilCycleCallback)
	return dec.mc.LastResult, err
}
