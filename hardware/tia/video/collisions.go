package video

import (
	"gopher2600/hardware/memory"
	"gopher2600/hardware/memory/vcssymbols"
)

type collisions struct {
	cxm0p  uint8
	cxm1p  uint8
	cxp0fb uint8
	cxp1fb uint8
	cxm0fb uint8
	cxm1fb uint8
	cxblpf uint8
	cxppmm uint8
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
}

// NOTE that collisions are detected in the video.Pixel() command

func (col *collisions) SetMemory(mem memory.ChipBus) {
	mem.ChipWrite(vcssymbols.CXM0P, col.cxm0p)
	mem.ChipWrite(vcssymbols.CXM1P, col.cxm1p)
	mem.ChipWrite(vcssymbols.CXP0FB, col.cxp0fb)
	mem.ChipWrite(vcssymbols.CXP1FB, col.cxp1fb)
	mem.ChipWrite(vcssymbols.CXM0FB, col.cxm0fb)
	mem.ChipWrite(vcssymbols.CXM1FB, col.cxm1fb)
	mem.ChipWrite(vcssymbols.CXBLPF, col.cxblpf)
	mem.ChipWrite(vcssymbols.CXPPMM, col.cxppmm)
}
