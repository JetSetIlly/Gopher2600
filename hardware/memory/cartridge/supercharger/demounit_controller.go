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

package supercharger

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jetsetilly/gopher2600/environment"
	"github.com/jetsetilly/gopher2600/hardware/memory/chipbus"
	"github.com/jetsetilly/gopher2600/hardware/memory/cpubus"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports"
	"github.com/jetsetilly/gopher2600/hardware/riot/ports/plugging"
	"github.com/jetsetilly/gopher2600/logger"
)

// eight ROM dumps of 8192 bytes each
const demoUnitRomCount = 8

var demoUnitRoms = [demoUnitRomCount]string{
	"Starpath Demonstration Unit 1P.bin",
	"Starpath Demonstration Unit 2P.bin",
	"Starpath Demonstration Unit 3P.bin",
	"Starpath Demonstration Unit 4P.bin",
	"Starpath Demonstration Unit 5P.bin",
	"Starpath Demonstration Unit 6P.bin",
	"Starpath Demonstration Unit 7P.bin",
	"Starpath Demonstration Unit 8P.bin",
}

type demoUnitState int

const (
	demoUnitSync demoUnitState = iota
	demoUnitTransfer
	demoUnitBlock
)

func (s demoUnitState) String() string {
	switch s {
	case demoUnitSync:
		return "sync"
	case demoUnitTransfer:
		return "transfer"
	case demoUnitBlock:
		return "block"
	}
	panic(fmt.Sprintf("unknown demoUnitState (%d)", s))
}

const (
	// each ROM fump in the list demoUnitRoms are divided into four 2048 byte banks
	demoUnitBankSize = 2048

	// the demo unit sections are loaded in transfer blocks of 64 bytes
	demoUnitBlockSize = 64
)

// each transfer block is interspersed with a 4 bit indicator (sent on the low nibble signal)
// indicating whether the section has more blocks to come or not
const (
	demoUnitContinue uint8 = 0x00
	demoUnitStop     uint8 = 0x0f
)

type demoUnit_controller struct {
	env *environment.Environment
	id  plugging.PortID
	bus ports.PeripheralBus

	transferTable []uint8
	schweber      []uint8
	data          []uint8

	hold int

	state demoUnitState
	sync  int

	waitForHi bool

	tableIdx int
	transfer section
}

type section struct {
	entry     int
	bank      int
	startAddr uint16
	blocks    int
	delay     int
	data      []uint8
	idx       int
	short     bool
}

func (t section) isEnded() bool {
	return t.idx >= len(t.data)-1
}

func (t section) String() string {
	var s strings.Builder
	if n, ok := demoUnitSections[t.entry]; ok {
		fmt.Fprintf(&s, "'%s'", n)
	} else {
		fmt.Fprintf(&s, "%d", t.entry)
	}
	fmt.Fprintf(&s, ": %d blocks (%d bytes) from %04x of bank %d, delay %d",
		t.blocks, len(t.data), t.startAddr, t.bank, t.delay)
	return s.String()

}

// descriptions taken from SDEMCTRL.ASM
var demoUnitSections = map[int]string{
	0:   "count down timer",
	6:   "starpath logo",
	12:  "firework thing",
	18:  "play advanced games",
	23:  "play advanced games",
	29:  "phaser patrol demo",
	35:  "p.p. try it",
	41:  "p.p. game proper",
	47:  "supercharger upgrades",
	53:  "expands RAM",
	59:  "mindmaster plug",
	64:  "mindmaster plug",
	70:  "mindmaster demo",
	76:  "multiload plug",
	82:  "dragonstomper plug",
	88:  "dragonstomper demo",
	94:  "commie mutants plug",
	100: "commie mutants demo",
	106: "c.m. try it",
	111: "c.m. try it",
	117: "c.m. game proper",
	122: "c.m. game proper",
	128: "games on cass/cheap",
	134: "play killer sat/s.m.",
	140: "fireball plug",
	146: "fireball demo",
	152: "use w/ any tape player",
	158: "electronic games plug",
	163: "electronic games plug",
	168: "electronic games plug",
}

func newDemoUnitController(env *environment.Environment, id plugging.PortID, bus ports.PeripheralBus, schweber []uint8) ports.Peripheral {
	con := &demoUnit_controller{
		env:           env,
		id:            id,
		bus:           bus,
		transferTable: schweber[0x200:0x2b0],
		schweber:      schweber[:],
	}

	if id != plugging.PortRight {
		logger.Log(env, "demo unit", "can only be plugged into the right player port")
		return nil
	}

	pth := filepath.Dir(env.Loader.Filename)

	for i := range demoUnitRomCount {
		f := filepath.Join(pth, demoUnitRoms[i])
		d, err := os.ReadFile(f)
		if err != nil {
			logger.Log(env, "demo unit", err.Error())
			return nil
		}
		if len(d) != 8192 {
			logger.Logf(env, "demo unit", "%s is not the correct size (should be 8192 bytes)", demoUnitRoms[i])
			return nil
		}
		for n := range len(d) / demoUnitBankSize {
			con.data = append(con.data, d[n*demoUnitBankSize:(n+1)*demoUnitBankSize]...)
		}
	}

	// indicate that the console program can continue
	con.write(0x00)

	return con
}

func (con *demoUnit_controller) String() string {
	return con.transfer.String()
}

func (con *demoUnit_controller) Reset() {
}

func (con *demoUnit_controller) nextTransfer() bool {
	if con.transfer.idx < len(con.transfer.data)-1 {
		return false
	}

	const transferTableEntrySize = 6

	bank := int(con.transferTable[con.tableIdx])
	startAddr := (int(con.transferTable[con.tableIdx+1]) << 8) | int(con.transferTable[con.tableIdx+2])
	blocks := int(con.transferTable[con.tableIdx+3])
	delay := int(con.transferTable[con.tableIdx+4]) & 0x3f
	short := con.transferTable[con.tableIdx+5] != 0xff

	con.transfer = section{
		entry:     con.tableIdx,
		bank:      bank,
		startAddr: uint16(startAddr),
		blocks:    blocks,
		delay:     delay,
		short:     short,
	}

	if bank == 0x7f {
		// schweber chip
		idx := startAddr - 0x1800

		// cloning slices of the data because we will be reversing the blocks below
		con.transfer.data = slices.Clone(con.schweber[idx : idx+(blocks*demoUnitBlockSize)])
	} else {
		// other ROM chips
		idx := startAddr - 0x1000
		idx += bank * demoUnitBankSize

		// cloning slices of the data because we will be reversing the blocks below
		con.transfer.data = slices.Clone(con.data[idx : idx+(blocks*demoUnitBlockSize)])
	}

	// reverse blocks
	for b := range blocks {
		i := b * demoUnitBlockSize
		slices.Reverse(con.transfer.data[i : i+demoUnitBlockSize])
	}

	logger.Logf(con.env, "demo unit", "%s", con.transfer)

	// print information about the transfer and advance tableIdx ready for the next transfer
	if short {
		con.tableIdx += transferTableEntrySize - 1
	} else {
		con.tableIdx += transferTableEntrySize
	}

	// loop back to beginning
	if con.tableIdx >= len(con.transferTable) {
		con.tableIdx = 0
	}
	if con.transferTable[con.tableIdx] == 0xff {
		con.tableIdx = 0
	}

	return true
}

func (con *demoUnit_controller) Unplug() {
}

func (con *demoUnit_controller) Snapshot() ports.Peripheral {
	n := *con
	return &n
}

func (con *demoUnit_controller) Plumb(bus ports.PeripheralBus) {
	con.bus = bus
}

func (con *demoUnit_controller) PortID() plugging.PortID {
	return con.id
}

func (con *demoUnit_controller) ID() plugging.PeripheralID {
	return plugging.PeripheralID("Starpath Demo Unit")
}

func (con *demoUnit_controller) HandleEvent(ports.Event, ports.EventData) (bool, error) {
	return false, nil
}

func (con *demoUnit_controller) write(value uint8) {
	swcha := (value & 0x0e) << 3
	inpt5 := (value & 0x01) << 7
	con.bus.WriteSWCHx(con.id, 0x80|swcha)
	con.bus.WriteINPTx(chipbus.INPT5, inpt5)

	// a hold of 42 is approximately 12µs assuming a NTSC clock
	con.hold = 42
}

func (con *demoUnit_controller) Update(data chipbus.ChangedRegister) bool {
	switch data.Register {
	case cpubus.SWCHA:
		v := data.Value & 0x0f

		switch con.state {
		case demoUnitSync:
			if con.waitForHi {
				if v == 0x08 {
					con.waitForHi = false
					if con.sync <= 4 {
						con.state = demoUnitTransfer
						con.nextTransfer()
						con.sync = 0
					}
				}
			} else {
				if v == 0x00 {
					con.waitForHi = true
					con.sync = 0
				}
			}

		case demoUnitTransfer:
			if con.waitForHi {
				if v == 0x08 {
					con.waitForHi = false

					// transmit high nibble of first byte
					con.write((^con.transfer.data[con.transfer.idx]) >> 4)

					// continue with demoUnitBlock state
					con.state = demoUnitBlock
				}
			} else {
				if v == 0x00 {
					con.waitForHi = true
					if con.transfer.isEnded() {
						if con.transfer.short {
							con.write(demoUnitContinue)
							con.nextTransfer()
						} else {
							con.state = demoUnitSync
						}
					} else {
						con.write(demoUnitContinue)
					}
				}
			}

		case demoUnitBlock:
			if con.waitForHi {
				if v == 0x08 {
					con.waitForHi = false

					// write infoByte every 'demoUnitBlockSize' bytes
					// OR high nibble of data block
					if con.transfer.idx%demoUnitBlockSize == 0 {
						con.state = demoUnitTransfer

						// apply delay as appropriate
						if con.transfer.isEnded() {
							if con.transfer.short {
								con.write(demoUnitContinue)
							} else {
								con.write(demoUnitStop)
								if con.transfer.delay > 0 {
									con.hold = con.transfer.delay * 2000000
								}
							}
						} else {
							con.write(demoUnitContinue)
						}
					} else {
						con.write((^con.transfer.data[con.transfer.idx]) >> 4)
					}
				}
			} else {
				if v == 0x00 {
					con.waitForHi = true

					// low nibble
					con.write(^con.transfer.data[con.transfer.idx] & 0x0f)
					con.transfer.idx++
				}
			}
		}
	}

	return false
}

func (con *demoUnit_controller) Step() {
	if con.hold > 0 {
		con.hold--
		if con.hold == 0 {
			con.write(demoUnitContinue)
		}
	}
	if con.state == demoUnitSync {
		con.sync++
	}
}

func (con *demoUnit_controller) IsActive() bool {
	return true
}
