package video

import (
	"fmt"
	"gopher2600/hardware/memory"
	"gopher2600/hardware/memory/vcssymbols"
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
	col.mem.ChipWrite(vcssymbols.CXM0P, 0)
	col.mem.ChipWrite(vcssymbols.CXM1P, 0)
	col.mem.ChipWrite(vcssymbols.CXP0FB, 0)
	col.mem.ChipWrite(vcssymbols.CXP1FB, 0)
	col.mem.ChipWrite(vcssymbols.CXM0FB, 0)
	col.mem.ChipWrite(vcssymbols.CXM1FB, 0)
	col.mem.ChipWrite(vcssymbols.CXBLPF, 0)
	col.mem.ChipWrite(vcssymbols.CXPPMM, 0)
}

// NOTE that collisions are detected in the video.Pixel() command

func (col *collisions) SetMemory(collisionAddress uint16) {
	switch collisionAddress {
	case vcssymbols.CXM0P:
		col.mem.ChipWrite(vcssymbols.CXM0P, col.cxm0p)
	case vcssymbols.CXM1P:
		col.mem.ChipWrite(vcssymbols.CXM1P, col.cxm1p)
	case vcssymbols.CXP0FB:
		col.mem.ChipWrite(vcssymbols.CXP0FB, col.cxp0fb)
	case vcssymbols.CXP1FB:
		col.mem.ChipWrite(vcssymbols.CXP1FB, col.cxp1fb)
	case vcssymbols.CXM0FB:
		col.mem.ChipWrite(vcssymbols.CXM0FB, col.cxm0fb)
	case vcssymbols.CXM1FB:
		col.mem.ChipWrite(vcssymbols.CXM1FB, col.cxm1fb)
	case vcssymbols.CXBLPF:
		col.mem.ChipWrite(vcssymbols.CXBLPF, col.cxblpf)
	case vcssymbols.CXPPMM:
		col.mem.ChipWrite(vcssymbols.CXPPMM, col.cxppmm)
	default:
		panic(fmt.Errorf("unkown collision address (%04x)", collisionAddress))
	}
}
