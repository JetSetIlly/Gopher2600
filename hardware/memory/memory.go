package memory

import (
	"fmt"
)

// Bus defines the operations for a memory system
type Bus interface {
	Clear()
	Read(address uint16) (uint8, error)
	Write(address uint16, data uint8) error
}

// VCS is the implementation of Memory for the VCS
type VCS struct {
	data           []uint8
	currentAddress uint16
}

func (mem *VCS) String() string {
	s := "0000  "
	i := 0
	j := 16
	for _, d := range mem.data {
		s = fmt.Sprintf("%s%02x ", s, d)
		i++
		if i == 16 {
			s = fmt.Sprintf("%s\n%04d  ", s, j)
			i = 0
			j += 16
		}
	}
	return s
}

// NewVCSMemory is the preferred method of initialisation for VCSMemory
func NewVCSMemory() *VCS {
	mem := new(VCS)
	mem.data = make([]uint8, 65536)
	return mem
}

// Clear sets all bytes in memory to zero
func (mem *VCS) Clear() {
	for i := 0; i < len(mem.data); i++ {
		mem.data[i] = 0x00
	}
}

func (mem *VCS) Read(address uint16) (uint8, error) {
	if int(address) > len(mem.data) {
		return 0, fmt.Errorf("address out of range (%d)", address)
	}
	return mem.data[address], nil
}

func (mem *VCS) Write(address uint16, data uint8) error {
	if int(address) > len(mem.data) {
		return fmt.Errorf("address out of range (%d)", address)
	}
	mem.data[address] = data
	return nil
}
