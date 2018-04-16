package memory

import "fmt"

// MemoryMap returns the VCS memory map as a string
func (mem *VCSMemory) MemoryMap() string {

	s := fmt.Sprintf("VCS Memory Map\n----------\n")

	var area string
	var sr, er uint16

	for er = 0; er <= 0x1fff; er++ {
		_, a := mem.MapAddress(er)
		if a != area {
			if area != "" {
				s = fmt.Sprintf("%s%04x -> %04x\t%s\n", s, sr, er-uint16(1), area)
			}
			area = a
			sr = er
		}
	}
	if area != "" {
		s = fmt.Sprintf("%s%04x -> %04x\t%s\n", s, sr, er-uint16(1), area)
	}

	return s
}
