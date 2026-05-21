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

package elf

import (
	"os"

	"github.com/jetsetilly/gopher2600/logger"
	"github.com/jetsetilly/gopher2600/resources"
)

// the resource path to the nvram files
const elf_nvram = "elf_nvram"

type nonVolatileSection struct {
	size uint16
	addr uint32
}

type nonVolatileMem struct {
	initialised bool
	sections    [2]nonVolatileSection
}

// Called once to set size of each slot and pointer to slot buffers. If saved data already exists, the
// buffers will be initialized with the most recent data for each slot. If no save data exists yet,
// the buffers will be left unchanged.
// void vcsInitNvStore(uint16_t size0, uint16_t size1, uint8_t *p0, uint8_t *p1);
func vcsInitNvStore(mem *elfMemory) {
	mem.nv.initialised = true

	mem.nv.sections[0].size = uint16(mem.strongarm.running.registers[0])
	mem.nv.sections[1].size = uint16(mem.strongarm.running.registers[1])
	mem.nv.sections[0].addr = mem.strongarm.running.registers[2]
	mem.nv.sections[1].addr = mem.strongarm.running.registers[3]

	p, err := resources.JoinPath(elf_nvram, mem.env.Loader.HashMD5)
	if err != nil {
		logger.Log(mem.env, "ELF", err)
		return
	}

	st, err := os.Stat(p)
	if err != nil {
		logger.Log(mem.env, "ELF", err)
		return
	}

	sz := int64(mem.nv.sections[0].size+mem.nv.sections[1].size) * 4
	if st.Size() != sz {
		logger.Logf(mem.env, "ELF", "%s is not %d bytes in size", p, sz)
		return
	}

	f, err := os.Open(p)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Log(mem.env, "ELF", err)
		}
		return
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logger.Log(mem.env, "ELF", err)
		}
	}()

	buf := make([]byte, sz*4)
	f.Read(buf)

	for _, sec := range mem.nv.sections {
		data, origin := mem.MapAddress(sec.addr, true, false)
		if data == nil {
			logger.Logf(mem.env, "ELF", "cannot find memory for address %#08x", sec.addr)
			return
		}
		copy((*data)[sec.addr-origin:], buf[:sec.size])
	}
}

// Can be called anytime including when a write is already in progress
// false = slot 0, true = slot 1
// void vcsWriteNvChunk(bool index);
func vcsWriteNvChunk(mem *elfMemory) {
	if !mem.nv.initialised {
		logger.Log(mem.env, "ELF", "NV memory is not initialised")
		return
	}

	p, err := resources.JoinPath(elf_nvram, mem.env.Loader.HashMD5)
	if err != nil {
		logger.Log(mem.env, "ELF", err)
		return
	}

	f, err := os.Create(p)
	if err != nil {
		logger.Log(mem.env, "ELF", err)
		return
	}
	defer func() {
		err := f.Close()
		if err != nil {
			logger.Log(mem.env, "ELF", err)
		}
	}()

	for _, sec := range mem.nv.sections {
		data, origin := mem.MapAddress(sec.addr, true, false)
		if data == nil {
			logger.Logf(mem.env, "ELF", "cannot find memory for address %#08x", sec.addr)
			return
		}
		n, err := f.Write((*data)[sec.addr-origin : sec.addr+uint32(sec.size*4)-origin])
		if err != nil {
			logger.Log(mem.env, "ELF", err)
			return
		}
		if n != int(sec.size*4) {
			logger.Log(mem.env, "ELF", "incorrect number of bytes written to NV file")
			return
		}
	}
}

// Call this once per frame, Ideally soon after vcsEndOverblank() just after a sta3(wsync) of nop2n()
// Writing to the flash will happen here, so it's best called from functions running out of RAM
// void vcsProcessNvStoreEvents();
func vcsProcessNvStoreEvents(mem *elfMemory) {
	if !mem.nv.initialised {
		logger.Log(mem.env, "ELF", "NV memory is not initialised")
		return
	}
}

// Returns true when there are writes queued or in progress. An indicatore should be shown to
// players so they do not power off the system while a write is in progress. Carts that write to local
// flash may only take a few frames to complete. Carts that save to cloud may take longer.
// bool vcsIsNvBusy();
func vcsIsNvBusy(mem *elfMemory) {
	if !mem.nv.initialised {
		logger.Log(mem.env, "ELF", "NV memory is not initialised")
		return
	}
	_ = mem.arm.RegisterSet(0, 0)
}
