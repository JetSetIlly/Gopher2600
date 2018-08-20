package memory

import "fmt"

// MemoryMap returns the VCS memory map as a string
func (mem VCSMemory) MemoryMap() string {
	var mm string
	var areaLabel string
	var sr, er uint16

	for er = 0; er <= 0x1fff; er++ {
		area := mem.memmap[mem.MapAddress(er, true)]

		if area.Label() != areaLabel {
			if areaLabel != "" {
				mm = fmt.Sprintf("%s%04x -> %04x\t%s\n", mm, sr, er-uint16(1), areaLabel)
			}
			areaLabel = area.Label()
			sr = er
		}
	}
	if areaLabel != "" {
		mm = fmt.Sprintf("%s%04x -> %04x\t%s\n", mm, sr, er-uint16(1), areaLabel)
	}

	return mm
}
