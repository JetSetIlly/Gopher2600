package video

import (
	"fmt"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/memory/addresses"
)

type collisions struct {
	mem memory.ChipBus

	cxm0p  uint8
	cxm1p  uint8
	cxp0fb uint8
	cxp1fb uint8
	cxm0fb uint8
	cxm1fb uint8
	cxblpf uint8
	cxppmm uint8
}

func newCollision(mem memory.ChipBus) *collisions {
	col := new(collisions)
	col.mem = mem
	return col
}

func (col *collisions) clear() {
	col.cxm0p = 0
	col.cxm1p = 0
	col.cxp0fb = 0
	col.cxp1fb = 0
	col.cxm0fb = 0
	col.cxm1fb = 0
	col.cxblpf = 0
	col.cxppmm = 0
	col.mem.ChipWrite(addresses.CXM0P, 0)
	col.mem.ChipWrite(addresses.CXM1P, 0)
	col.mem.ChipWrite(addresses.CXP0FB, 0)
	col.mem.ChipWrite(addresses.CXP1FB, 0)
	col.mem.ChipWrite(addresses.CXM0FB, 0)
	col.mem.ChipWrite(addresses.CXM1FB, 0)
	col.mem.ChipWrite(addresses.CXBLPF, 0)
	col.mem.ChipWrite(addresses.CXPPMM, 0)
}

func (col *collisions) SetMemory(collisionAddress uint16) {
	switch collisionAddress {
	case addresses.CXM0P:
		col.mem.ChipWrite(addresses.CXM0P, col.cxm0p)
	case addresses.CXM1P:
		col.mem.ChipWrite(addresses.CXM1P, col.cxm1p)
	case addresses.CXP0FB:
		col.mem.ChipWrite(addresses.CXP0FB, col.cxp0fb)
	case addresses.CXP1FB:
		col.mem.ChipWrite(addresses.CXP1FB, col.cxp1fb)
	case addresses.CXM0FB:
		col.mem.ChipWrite(addresses.CXM0FB, col.cxm0fb)
	case addresses.CXM1FB:
		col.mem.ChipWrite(addresses.CXM1FB, col.cxm1fb)
	case addresses.CXBLPF:
		col.mem.ChipWrite(addresses.CXBLPF, col.cxblpf)
	case addresses.CXPPMM:
		col.mem.ChipWrite(addresses.CXPPMM, col.cxppmm)
	default:
		panic(fmt.Sprintf("not a collision address (%04x)", collisionAddress))
	}
}
