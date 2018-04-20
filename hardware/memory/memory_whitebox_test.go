package memory

import "testing"

func TestCartridge(t *testing.T) {
	cart := NewCart()
	if bankSize != cart.memtop-cart.origin+1 {
		t.Errorf("cartridge bank size and/or memtop/origin incorrectly defined")
	}
}
