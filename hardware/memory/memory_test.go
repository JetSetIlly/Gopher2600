package memory_test

import (
	"gopher2600/hardware/memory"
	"testing"
)

const (
	validMemMap = "0000 -> 007f	TIA\n0080 -> 00ff	PIA RAM\n0100 -> 017f	TIA\n0180 -> 01ff	PIA RAM\n0200 -> 027f	TIA\n0280 -> 02ff	RIOT\n0300 -> 037f	TIA\n0380 -> 03ff	RIOT\n0400 -> 047f	TIA\n0480 -> 04ff	PIA RAM\n0500 -> 057f	TIA\n0580 -> 05ff	PIA RAM\n0600 -> 067f	TIA\n0680 -> 06ff	RIOT\n0700 -> 077f	TIA\n0780 -> 07ff	RIOT\n0800 -> 087f	TIA\n0880 -> 08ff	PIA RAM\n0900 -> 097f	TIA\n0980 -> 09ff	PIA RAM\n0a00 -> 0a7f	TIA\n0a80 -> 0aff	RIOT\n0b00 -> 0b7f	TIA\n0b80 -> 0bff	RIOT\n0c00 -> 0c7f	TIA\n0c80 -> 0cff	PIA RAM\n0d00 -> 0d7f	TIA\n0d80 -> 0dff	PIA RAM\n0e00 -> 0e7f	TIA\n0e80 -> 0eff	RIOT\n0f00 -> 0f7f	TIA\n0f80 -> 0fff	RIOT\n1000 -> 1fff	Cartridge\n"
)

func TestMemory(t *testing.T) {
	mem, err := memory.NewVCSMemory()
	if err != nil {
		t.Fatalf(err.Error())
	}

	mem.Clear()

	if mem.MemoryMap() != validMemMap {
		t.Fatalf("memory map is invalid")
	}
}
