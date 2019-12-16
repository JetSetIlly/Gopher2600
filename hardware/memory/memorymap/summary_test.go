package memorymap_test

import (
	"gopher2600/hardware/memory/memorymap"
	"testing"
)

const validMemMap = `0000 -> 007f	TIA
0080 -> 00ff	RAM
0100 -> 017f	TIA
0180 -> 01ff	RAM
0200 -> 027f	TIA
0280 -> 02ff	RIOT
0300 -> 037f	TIA
0380 -> 03ff	RIOT
0400 -> 047f	TIA
0480 -> 04ff	RAM
0500 -> 057f	TIA
0580 -> 05ff	RAM
0600 -> 067f	TIA
0680 -> 06ff	RIOT
0700 -> 077f	TIA
0780 -> 07ff	RIOT
0800 -> 087f	TIA
0880 -> 08ff	RAM
0900 -> 097f	TIA
0980 -> 09ff	RAM
0a00 -> 0a7f	TIA
0a80 -> 0aff	RIOT
0b00 -> 0b7f	TIA
0b80 -> 0bff	RIOT
0c00 -> 0c7f	TIA
0c80 -> 0cff	RAM
0d00 -> 0d7f	TIA
0d80 -> 0dff	RAM
0e00 -> 0e7f	TIA
0e80 -> 0eff	RIOT
0f00 -> 0f7f	TIA
0f80 -> 0fff	RIOT
1000 -> 1fff	Cartridge
`

func TestMemory(t *testing.T) {
	if memorymap.Summary() != validMemMap {
		t.Fatalf("memory map is invalid")
	}
}
