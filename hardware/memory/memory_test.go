package memory_test

import (
	"gopher2600/hardware/memory"
	"testing"
)

func readData(t *testing.T, mem *memory.VCSMemory, address uint16, expectedData uint8) {
	t.Helper()
	d, err := mem.Read(address)
	if err != nil {
		t.Errorf("unexpected error (%s)", err)
	}
	if d != expectedData {
		t.Errorf("expecting %#02x received %#02x", expectedData, d)
	}
}

func TestDataMask(t *testing.T) {
	mem, err := memory.NewVCSMemory()
	if err != nil {
		t.Errorf("unexpected error (%s)", err)
	}

	// no data in register
	readData(t, mem, 0x00, 0x00)
	readData(t, mem, 0x02, 0x02)
	readData(t, mem, 0x171, 0x01)

	// hight bits set in register
	mem.TIA.ChipWrite(0x00, 0x40)
	readData(t, mem, 0x00, 0x40)
	mem.TIA.ChipWrite(0x01, 0x80)
	readData(t, mem, 0x01, 0x81)

	// low bits set too. low bits in address supercede low bits in register
	mem.TIA.ChipWrite(0x01, 0x8f)
	readData(t, mem, 0x01, 0x81)
	mem.TIA.ChipWrite(0x02, 0xc7)
	readData(t, mem, 0x02, 0xc2)

	// non-zero-page addressing
	readData(t, mem, 0x171, 0x81)
}
